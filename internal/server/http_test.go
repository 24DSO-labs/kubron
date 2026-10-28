// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"testing/fstest"

	"github.com/VadimIl224/kubron/internal/auth"
)

type httpTarget struct{}

func (httpTarget) Add(a, b int) (int, error) { return a + b, nil }
func (httpTarget) NeedLogin() error          { return auth.ErrNotAuthenticated }
func (httpTarget) NeedRight() error          { return fmt.Errorf("обёрнуто: %w", auth.ErrForbidden) }
func (httpTarget) Plain() error              { return errors.New("что-то пошло не так") }

func testHandler(t *testing.T) http.Handler {
	t.Helper()
	d, err := NewDispatcher(httpTarget{}, []string{"Add", "NeedLogin", "NeedRight", "Plain"},
		map[string]string{"Hidden": "закрыт"})
	if err != nil {
		t.Fatal(err)
	}
	static := fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/a-1.js": {Data: []byte("console.log(1)")},
	}
	return NewHandler(d, static, nil)
}

type rpcReply struct {
	Result json.RawMessage `json:"result"`
	Error  *rpcError       `json:"error"`
}

func post(h http.Handler, path, body string, hdr map[string]string) (*httptest.ResponseRecorder, rpcReply) {
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	for k, v := range hdr {
		req.Header.Set(k, v)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var rep rpcReply
	_ = json.Unmarshal(rec.Body.Bytes(), &rep)
	return rec, rep
}

func TestRPCSuccessAndErrorMapping(t *testing.T) {
	h := testHandler(t)

	rec, rep := post(h, "/api/rpc/Add", "[2,3]", nil)
	if rec.Code != 200 || string(rep.Result) != "5" {
		t.Fatalf("Add: %d %s", rec.Code, rec.Body)
	}
	for _, c := range []struct {
		path   string
		body   string
		status int
		code   string
	}{
		{"/api/rpc/NeedLogin", "[]", 401, "unauthenticated"},
		{"/api/rpc/NeedRight", "[]", 403, "forbidden"},
		{"/api/rpc/Plain", "[]", 500, "error"},
		{"/api/rpc/Missing", "[]", 404, "unknown_method"},
		{"/api/rpc/Hidden", "[]", 403, "method_denied"},
		{"/api/rpc/Add", "[1]", 400, "bad_request"},
		{"/api/rpc/Add", "{}", 400, "bad_request"},
	} {
		rec, rep := post(h, c.path, c.body, nil)
		if rec.Code != c.status || rep.Error == nil || rep.Error.Code != c.code {
			t.Errorf("%s %s: статус %d, ответ %s; ждали %d/%s", c.path, c.body, rec.Code, rec.Body, c.status, c.code)
		}
	}
	// сообщение об ошибке сохраняется, чтобы фронтенд распознал «требуется вход в систему»
	_, rep = post(h, "/api/rpc/NeedLogin", "[]", nil)
	if rep.Error.Message != auth.ErrNotAuthenticated.Error() {
		t.Errorf("сообщение изменено: %q", rep.Error.Message)
	}
}

func TestRPCRejectsForeignOriginAndWrongContentType(t *testing.T) {
	h := testHandler(t)

	rec, _ := post(h, "/api/rpc/Add", "[1,2]", map[string]string{"Origin": "https://evil.example"})
	if rec.Code != 403 {
		t.Errorf("чужой Origin: %d", rec.Code)
	}
	rec, rep := post(h, "/api/rpc/Add", "[1,2]", map[string]string{"Origin": "http://example.com"})
	if rec.Code != 200 || string(rep.Result) != "3" {
		t.Errorf("тот же Origin (Host=example.com) должен проходить: %d %s", rec.Code, rec.Body)
	}
	rec, _ = post(h, "/api/rpc/Add", "[1,2]", map[string]string{"Content-Type": "text/plain"})
	if rec.Code != 415 {
		t.Errorf("text/plain (CSRF без preflight) должен отклоняться: %d", rec.Code)
	}
	rec, _ = post(h, "/api/rpc/Add", "[1,2]", map[string]string{"Content-Type": "application/json; charset=utf-8"})
	if rec.Code != 200 {
		t.Errorf("application/json с charset должен проходить: %d", rec.Code)
	}
}

func TestRPCBodyLimit(t *testing.T) {
	h := testHandler(t)
	big := "[\"" + strings.Repeat("a", maxRPCBody+10) + "\"]"
	rec, _ := post(h, "/api/rpc/Add", big, nil)
	if rec.Code != http.StatusRequestEntityTooLarge {
		t.Errorf("слишком большое тело: %d", rec.Code)
	}
}

func get(h http.Handler, path string) *httptest.ResponseRecorder {
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, path, nil))
	return rec
}

func TestStaticAndSPAFallback(t *testing.T) {
	h := testHandler(t)

	if rec := get(h, "/"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "app") {
		t.Errorf("/: %d %s", rec.Code, rec.Body)
	}
	rec := get(h, "/assets/a-1.js")
	if rec.Code != 200 || !strings.Contains(rec.Header().Get("Cache-Control"), "immutable") {
		t.Errorf("asset: %d %v", rec.Code, rec.Header())
	}
	rec = get(h, "/clusters/prod/pods") // клиентский маршрут
	if rec.Code != 200 || !strings.Contains(rec.Body.String(), "app") || rec.Header().Get("Cache-Control") != "no-cache" {
		t.Errorf("SPA-fallback: %d %s %v", rec.Code, rec.Body, rec.Header())
	}
	if rec := get(h, "/api/unknown"); rec.Code != 404 || !strings.Contains(rec.Header().Get("Content-Type"), "json") {
		t.Errorf("/api/unknown должен отдавать JSON 404, а не index.html: %d %s", rec.Code, rec.Body)
	}
	if rec := get(h, "/healthz"); rec.Code != 200 {
		t.Errorf("healthz: %d", rec.Code)
	}
	// ServeMux нормализует путь редиректом (301/307 — зависит от версии Go); в любом случае содержимое вне корня отдаваться не должно
	if rec := get(h, "/../../etc/passwd"); (rec.Code != 200 && (rec.Code < 300 || rec.Code > 399)) || strings.Contains(rec.Body.String(), "root:") {
		t.Errorf("выход за корень: %d", rec.Code)
	}
	if rec := get(h, "/etc/passwd"); rec.Code != 200 || !strings.Contains(rec.Body.String(), "app") {
		t.Errorf("несуществующий файл должен отдавать SPA-fallback: %d", rec.Code)
	}
	if h := get(h, "/").Header(); h.Get("X-Content-Type-Options") != "nosniff" || h.Get("X-Frame-Options") != "DENY" {
		t.Errorf("заголовки безопасности: %v", h)
	}
}

func TestNoFrontendGivesClearMessage(t *testing.T) {
	d, _ := NewDispatcher(httpTarget{}, []string{"Add"}, nil)
	rec := get(NewHandler(d, nil, nil), "/")
	if rec.Code != http.StatusServiceUnavailable {
		t.Errorf("без фронтенда ждали 503: %d", rec.Code)
	}
	rec = get(NewHandler(d, fstest.MapFS{}, nil), "/")
	if rec.Code != http.StatusServiceUnavailable || !strings.Contains(rec.Body.String(), "index.html") {
		t.Errorf("пустой фронтенд: %d %s", rec.Code, rec.Body)
	}
}
