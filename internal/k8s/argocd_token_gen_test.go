// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"encoding/json"
	"io"
	"io/fs"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const (
	testPassword = "S3CRET-PASSWORD-123"
	testSession  = "S3CRET-SESSION-JWT"
	testAcctTok  = "issued.account.token"
)

// argoFake — минимальный Argo CD: /api/v1/session и /api/v1/account/{name}/token.
type argoFake struct {
	sessionStatus, tokenStatus int
	sessionBody, tokenBody     string
	sessionReq, tokenReq       map[string]any
	tokenPath, tokenAuth       string
}

func (f *argoFake) handler() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		var in map[string]any
		_ = json.Unmarshal(raw, &in)
		switch {
		case r.URL.Path == "/argocd/api/v1/session":
			f.sessionReq = in
			w.WriteHeader(f.sessionStatus)
			_, _ = w.Write([]byte(f.sessionBody))
		case strings.HasPrefix(r.URL.Path, "/argocd/api/v1/account/"):
			f.tokenReq, f.tokenPath, f.tokenAuth = in, r.URL.Path, r.Header.Get("Authorization")
			w.WriteHeader(f.tokenStatus)
			_, _ = w.Write([]byte(f.tokenBody))
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})
}

func newFake() *argoFake {
	return &argoFake{
		sessionStatus: 200, sessionBody: `{"token":"` + testSession + `"}`,
		tokenStatus: 200, tokenBody: `{"token":"` + testAcctTok + `"}`,
	}
}

func runGenerate(t *testing.T, f *argoFake, account, user, pass string, days int) (string, error) {
	t.Helper()
	srv := httptest.NewServer(f.handler())
	defer srv.Close()
	c, err := newArgoAPIClient(srv.URL+"/argocd", "")
	if err != nil {
		t.Fatal(err)
	}
	return generateArgoToken(context.Background(), c, account, user, pass, days)
}

func TestGenerateArgoTokenHappyPath(t *testing.T) {
	f := newFake()
	tok, err := runGenerate(t, f, "kubron", "admin", testPassword, 90)
	if err != nil {
		t.Fatal(err)
	}
	if tok != testAcctTok {
		t.Errorf("token = %q", tok)
	}
	if f.sessionReq["username"] != "admin" || f.sessionReq["password"] != testPassword {
		t.Errorf("session request: %v", f.sessionReq)
	}
	if f.tokenPath != "/argocd/api/v1/account/kubron/token" || f.tokenAuth != "Bearer "+testSession {
		t.Errorf("token request: path=%q auth=%q", f.tokenPath, f.tokenAuth)
	}
	if f.tokenReq["name"] != "kubron" || f.tokenReq["expiresIn"] != float64(90*24*3600) {
		t.Errorf("token body: %v", f.tokenReq)
	}
}

func TestGenerateArgoTokenDefaultExpiry(t *testing.T) {
	f := newFake()
	if _, err := runGenerate(t, f, "kubron", "admin", testPassword, 0); err != nil {
		t.Fatal(err)
	}
	if f.tokenReq["expiresIn"] != float64(argoTokenDefaultDays*24*3600) {
		t.Errorf("срок по умолчанию: %v", f.tokenReq["expiresIn"])
	}
}

func TestGenerateArgoTokenValidation(t *testing.T) {
	cases := []struct {
		name, account, user, pass string
		days                      int
	}{
		{"пустой аккаунт", "", "u", "p", 30},
		{"аккаунт с путём", "../admin", "u", "p", 30},
		{"аккаунт с пробелом", "a b", "u", "p", 30},
		{"нет логина", "kubron", "", "p", 30},
		{"нет пароля", "kubron", "u", "", 30},
		{"срок отрицательный", "kubron", "u", "p", -1},
		{"срок слишком большой", "kubron", "u", "p", argoTokenMaxDays + 1},
		{"слишком длинный пароль", "kubron", "u", strings.Repeat("x", argoCredMaxLen+1), 30},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			if _, err := runGenerate(t, f, tc.account, tc.user, tc.pass, tc.days); err == nil {
				t.Error("ожидалась ошибка валидации")
			}
			if f.sessionReq != nil {
				t.Error("запрос в сеть не должен уходить при неверном вводе")
			}
		})
	}
}

func TestGenerateArgoTokenErrorsDoNotLeakSecrets(t *testing.T) {
	cases := []struct {
		name  string
		mod   func(f *argoFake)
		want  string
		wantN int // сколько запросов дошло до сервера (0 — только сессия)
	}{
		{"неверный пароль", func(f *argoFake) {
			f.sessionStatus, f.sessionBody = 401, `{"message":"bad `+testPassword+`"}`
		}, "отклонил логин", 0},
		{"сессия 500 с паролем в тексте", func(f *argoFake) {
			f.sessionStatus, f.sessionBody = 500, `{"message":"oops `+testPassword+`"}`
		}, "HTTP 500", 0},
		{"нет apiKey", func(f *argoFake) {
			f.tokenStatus, f.tokenBody = 400, `{"message":"account 'kubron' does not have apiKey capability"}`
		}, "accounts.kubron: apiKey", 1},
		{"нет прав RBAC", func(f *argoFake) {
			f.tokenStatus, f.tokenBody = 403, `{"message":"permission denied `+testSession+`"}`
		}, "нет права создавать токены", 1},
		{"токен 500 с секретами в тексте", func(f *argoFake) {
			f.tokenStatus, f.tokenBody = 500, `{"message":"x `+testSession+` `+testPassword+`"}`
		}, "HTTP 500", 1},
		{"сессия без токена", func(f *argoFake) { f.sessionBody = `{}` }, "не вернул токен сессии", 0},
		{"аккаунт-токен пустой", func(f *argoFake) { f.tokenBody = `{"token":""}` }, "пустой или некорректный", 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := newFake()
			tc.mod(f)
			_, err := runGenerate(t, f, "kubron", "admin", testPassword, 30)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("ожидалась ошибка с %q, получили %v", tc.want, err)
			}
			for _, secret := range []string{testPassword, testSession, testAcctTok} {
				if strings.Contains(err.Error(), secret) {
					t.Errorf("секрет %q попал в текст ошибки: %v", secret, err)
				}
			}
			if (f.tokenReq != nil) != (tc.wantN == 1) {
				t.Errorf("запрос токена: ожидалось=%v, было=%v", tc.wantN == 1, f.tokenReq != nil)
			}
		})
	}
}

func TestScrubHandlesJSONEscapedSecrets(t *testing.T) {
	pass := `pa"ss\word`
	body, _ := json.Marshal(map[string]string{"message": "bad " + pass})
	got := string(scrub(body, pass))
	if strings.Contains(got, `pa\"ss`) || strings.Contains(got, "word") {
		t.Errorf("пароль не вычищен: %s", got)
	}
}

// Токен сохраняется для кластера и читается обратно; пароль в файл не попадает.
func TestGeneratedTokenIsStoredWithoutPassword(t *testing.T) {
	dir := useTempTokenDir(t)
	f := newFake()
	tok, err := runGenerate(t, f, "kubron", "admin", testPassword, 30)
	if err != nil {
		t.Fatal(err)
	}
	if err := saveArgoToken("c1", tok); err != nil {
		t.Fatal(err)
	}
	if got, err := loadArgoToken("c1"); err != nil || got != testAcctTok {
		t.Fatalf("load: %q %v", got, err)
	}
	assertNoSecretOnDisk(t, dir, testPassword, testSession)
}

func assertNoSecretOnDisk(t *testing.T, dir string, secrets ...string) {
	t.Helper()
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, rerr := os.ReadFile(path)
		if rerr != nil {
			return rerr
		}
		for _, s := range secrets {
			if strings.Contains(string(data), s) {
				t.Errorf("секрет %q найден в файле %s", s, path)
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}
