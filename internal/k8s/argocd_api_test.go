package k8s

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

const testAPIToken = "S3CRET-API-TOKEN"

func TestArgoAPIManagedResources(t *testing.T) {
	var gotAuth, gotPath, gotNS string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth, gotPath, gotNS = r.Header.Get("Authorization"), r.URL.Path, r.URL.Query().Get("appNamespace")
		_, _ = w.Write([]byte(`{"items":[{"group":"apps","kind":"Deployment","namespace":"prod","name":"web","hook":false,` +
			`"targetState":"{\"a\":1}","liveState":"null","normalizedLiveState":"","predictedLiveState":"{\"a\":2}"}]}`))
	}))
	defer srv.Close()

	c, err := newArgoAPIClient(srv.URL+"/argocd/", testAPIToken)
	if err != nil {
		t.Fatal(err)
	}
	items, err := c.managedResources(context.Background(), "argocd", "my-app")
	if err != nil {
		t.Fatal(err)
	}
	if gotAuth != "Bearer "+testAPIToken {
		t.Errorf("Authorization: %q", gotAuth)
	}
	if gotPath != "/argocd/api/v1/applications/my-app/managed-resources" || gotNS != "argocd" {
		t.Errorf("запрос: path=%q appNamespace=%q (rootpath должен сохраняться)", gotPath, gotNS)
	}
	if len(items) != 1 || items[0].Kind != "Deployment" || items[0].PredictedLiveState != `{"a":2}` {
		t.Errorf("items: %+v", items)
	}
}

func TestArgoAPIStatusErrorsDoNotLeakToken(t *testing.T) {
	cases := []struct {
		status int
		body   string
		want   string
	}{
		{401, `{"message":"` + testAPIToken + ` is bad"}`, "недействителен"}, // тело 401 игнорируется целиком
		{403, `{}`, "нет прав"},
		{404, `{}`, "не найдено"},
		{500, `{"message":"boom"}`, "HTTP 500: boom"},
		{502, `not json`, "HTTP 502"},
		{500, `{"message":"` + strings.Repeat("x", 500) + `"}`, "…"},
	}
	for _, tc := range cases {
		t.Run(http.StatusText(tc.status)+"/"+tc.want, func(t *testing.T) {
			srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			}))
			defer srv.Close()
			c, _ := newArgoAPIClient(srv.URL, testAPIToken)
			_, err := c.managedResources(context.Background(), "ns", "app")
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ожидалась ошибка с %q, получили %v", tc.want, err)
			}
			if strings.Contains(err.Error(), testAPIToken) {
				t.Errorf("токен попал в текст ошибки: %v", err)
			}
			if len(err.Error()) > 300 {
				t.Errorf("текст ошибки не ограничен: %d байт", len(err.Error()))
			}
		})
	}
}

func TestArgoAPIBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { _, _ = w.Write([]byte("<html>")) }))
	defer srv.Close()
	c, _ := newArgoAPIClient(srv.URL, "t")
	if _, err := c.managedResources(context.Background(), "ns", "app"); err == nil || !strings.Contains(err.Error(), "неожиданный ответ") {
		t.Errorf("err = %v", err)
	}
}

func TestArgoAPIDoesNotFollowRedirects(t *testing.T) {
	var hits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { hits.Add(1) }))
	defer other.Close()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL, http.StatusFound)
	}))
	defer srv.Close()

	c, _ := newArgoAPIClient(srv.URL, testAPIToken)
	_, err := c.managedResources(context.Background(), "ns", "app")
	if err == nil || !strings.Contains(err.Error(), "HTTP 302") {
		t.Fatalf("ожидалась ошибка HTTP 302, получили %v", err)
	}
	if hits.Load() != 0 {
		t.Error("клиент пошёл по редиректу: токен мог уйти на чужой адрес")
	}
}

func TestArgoAPISelfSignedTLSOnLoopback(t *testing.T) {
	srv := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte(`{"items":[]}`))
	}))
	defer srv.Close()
	c, err := newArgoAPIClient(srv.URL, "t") // 127.0.0.1 + самоподписанный сертификат argocd-server
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.managedResources(context.Background(), "ns", "app"); err != nil {
		t.Fatalf("loopback-туннель с самоподписанным сертификатом должен работать: %v", err)
	}
}

func TestNewArgoAPIClientValidation(t *testing.T) {
	for _, bad := range []string{"", "localhost:8080", "ftp://localhost", "http://", "://x"} {
		if _, err := newArgoAPIClient(bad, "t"); err == nil {
			t.Errorf("%q должен быть отклонён", bad)
		}
	}
	for host, want := range map[string]bool{
		"localhost": true, "127.0.0.1": true, "::1": true,
		"argocd.example.com": false, "10.0.0.5": false, "": false,
	} {
		if got := isLoopbackHost(host); got != want {
			t.Errorf("isLoopbackHost(%q) = %v, want %v", host, got, want)
		}
	}
}
