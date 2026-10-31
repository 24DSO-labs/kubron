// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"testing/fstest"
	"time"
)

func hardeningDispatcher(t *testing.T) *Dispatcher {
	t.Helper()
	d, err := NewDispatcher(httpTarget{}, []string{"Add", "NeedLogin"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	return d
}

func hardeningStatic() fstest.MapFS {
	return fstest.MapFS{
		"index.html":    {Data: []byte("<html>app</html>")},
		"assets/a-1.js": {Data: []byte("console.log(1)")},
	}
}

// Заголовки на всех видах ответов настоящего обработчика: страница, статика, API, health, страницы ошибок.
func TestNewHandlerSecurityHeadersTable(t *testing.T) {
	g := &Guard{Sessions: NewSessions(time.Hour, false), TrustProxy: true}
	withStatic := NewHandler(hardeningDispatcher(t), hardeningStatic(), g)
	noStatic := NewHandler(hardeningDispatcher(t), nil, g)
	for _, c := range []struct {
		name   string
		h      http.Handler
		method string
		path   string
		status int
	}{
		{"index", withStatic, http.MethodGet, "/", http.StatusOK},
		{"SPA route", withStatic, http.MethodGet, "/clusters/x/pods", http.StatusOK},
		{"asset", withStatic, http.MethodGet, "/assets/a-1.js", http.StatusOK},
		{"healthz", withStatic, http.MethodGet, "/healthz", http.StatusOK},
		{"session", withStatic, http.MethodGet, "/api/session", http.StatusOK},
		{"rpc", withStatic, http.MethodPost, "/api/rpc/Add", http.StatusUnsupportedMediaType},
		{"unknown api", withStatic, http.MethodGet, "/api/nothing", http.StatusNotFound},
		{"405 page", withStatic, http.MethodPost, "/", http.StatusMethodNotAllowed},
		{"503 no frontend", noStatic, http.MethodGet, "/", http.StatusServiceUnavailable},
	} {
		req := httptest.NewRequest(c.method, c.path, nil)
		req.Host = "kubron.example.com"
		rec := httptest.NewRecorder()
		c.h.ServeHTTP(rec, req)
		if rec.Code != c.status {
			t.Errorf("%s: status %d, want %d", c.name, rec.Code, c.status)
		}
		hd := rec.Header()
		if d := directives(hd.Get("Content-Security-Policy")); d["script-src"] != "'self'" || d["frame-ancestors"] != "'none'" {
			t.Errorf("%s: CSP %q", c.name, hd.Get("Content-Security-Policy"))
		}
		for _, name := range []string{"X-Content-Type-Options", "X-Frame-Options", "Referrer-Policy", "Permissions-Policy",
			"Cross-Origin-Opener-Policy", "Cross-Origin-Resource-Policy"} {
			if hd.Get(name) == "" {
				t.Errorf("%s: no %s", c.name, name)
			}
		}
		if hd.Get("Strict-Transport-Security") != "" {
			t.Errorf("%s: HSTS over plain HTTP", c.name)
		}
	}
}

func TestNewHandlerHSTSFollowsTheSameLogicAsSecureCookie(t *testing.T) {
	static := hardeningStatic()
	for _, c := range []struct {
		name       string
		trustProxy bool
		xfp        string
		want       bool
	}{
		{"behind a trusted TLS proxy", true, "https", true},
		{"proxy header without trust", false, "https", false},
		{"plain http", true, "", false},
	} {
		g := &Guard{Sessions: NewSessions(time.Hour, false), TrustProxy: c.trustProxy}
		h := NewHandler(hardeningDispatcher(t), static, g)
		req := httptest.NewRequest(http.MethodGet, "/", nil)
		req.Host = "kubron.example.com"
		if c.xfp != "" {
			req.Header.Set("X-Forwarded-Proto", c.xfp)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if got := rec.Header().Get("Strict-Transport-Security") != ""; got != c.want {
			t.Errorf("%s: HSTS present=%v, want %v", c.name, got, c.want)
		}
		if secure := g.secure(req); secure != c.want {
			t.Errorf("%s: Guard.secure=%v disagrees with HSTS", c.name, secure)
		}
	}
	// Без Guard (тесты, одиночный режим) HSTS нет вообще
	rec := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	req.Header.Set("X-Forwarded-Proto", "https")
	NewHandler(hardeningDispatcher(t), static, nil).ServeHTTP(rec, req)
	if rec.Header().Get("Strict-Transport-Security") != "" {
		t.Error("HSTS without a trusted proxy")
	}
}

func TestNewHandlerCSPReportOnlyOption(t *testing.T) {
	h := NewHandler(hardeningDispatcher(t), hardeningStatic(), nil, WithCSPReportOnly(true))
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/", nil))
	if rec.Header().Get("Content-Security-Policy") != "" || rec.Header().Get("Content-Security-Policy-Report-Only") == "" {
		t.Errorf("headers: %v", rec.Header())
	}
}

func anonRPC(h http.Handler, xff string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(http.MethodPost, "/api/rpc/Add", strings.NewReader("[1,2]"))
	req.Header.Set("Content-Type", "application/json")
	req.RemoteAddr = "192.0.2.10:4000"
	if xff != "" {
		req.Header.Set("X-Forwarded-For", xff)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func TestAnonymousRPCIsRateLimitedWith429(t *testing.T) {
	g := &Guard{Sessions: NewSessions(time.Hour, false), AnonLimiter: NewRateLimiter(0.01, 3)}
	h := NewHandler(hardeningDispatcher(t), nil, g)
	for i := 0; i < 3; i++ {
		if rec := anonRPC(h, ""); rec.Code != http.StatusUnauthorized {
			t.Fatalf("request %d: %d (anonymous Add is 401, not limited yet)", i, rec.Code)
		}
	}
	rec := anonRPC(h, "")
	if rec.Code != http.StatusTooManyRequests {
		t.Fatalf("status %d, want 429", rec.Code)
	}
	secs, err := strconv.Atoi(rec.Header().Get("Retry-After"))
	if err != nil || secs < 1 {
		t.Errorf("Retry-After: %q", rec.Header().Get("Retry-After"))
	}
	var rep rpcReply
	if err := json.Unmarshal(rec.Body.Bytes(), &rep); err != nil || rep.Error == nil {
		t.Fatalf("body %q: %v", rec.Body.String(), err)
	}
	if rep.Error.Code != "rate_limited" || rep.Error.Message != fmt.Sprintf("too many requests, try again in %d s", secs) {
		t.Errorf("error format: %+v", rep.Error)
	}
	if rec.Header().Get("Content-Security-Policy") == "" {
		t.Error("the 429 response must carry the security headers too")
	}
}

// Подделка X-Forwarded-For не даёт обойти лимит: ключ — адрес справа, добавленный доверенным прокси.
func TestRateLimitIsNotBypassedByForgedForwardedFor(t *testing.T) {
	g := &Guard{Sessions: NewSessions(time.Hour, false), TrustProxy: true, AnonLimiter: NewRateLimiter(0.01, 3)}
	h := NewHandler(hardeningDispatcher(t), nil, g)
	var limited int
	for i := 0; i < 8; i++ {
		if anonRPC(h, fmt.Sprintf("10.9.9.%d, 203.0.113.9", i)).Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 5 {
		t.Errorf("limited %d of 8, want 5: rotating the left XFF element must not give a fresh bucket", limited)
	}
	// Без доверия к прокси заголовок игнорируется совсем, ключом остаётся RemoteAddr
	g2 := &Guard{Sessions: NewSessions(time.Hour, false), AnonLimiter: NewRateLimiter(0.01, 3)}
	h2 := NewHandler(hardeningDispatcher(t), nil, g2)
	limited = 0
	for i := 0; i < 8; i++ {
		if anonRPC(h2, fmt.Sprintf("198.51.100.%d", i)).Code == http.StatusTooManyRequests {
			limited++
		}
	}
	if limited != 5 {
		t.Errorf("without proxy trust: limited %d of 8, want 5", limited)
	}
	// Другой клиент за тем же прокси не страдает
	if rec := anonRPC(h, "203.0.113.77"); rec.Code == http.StatusTooManyRequests {
		t.Error("a different client behind the proxy must have its own bucket")
	}
}

// Запросы с действующей сессией лимитом анонимных вызовов не затрагиваются.
func TestAuthenticatedRPCIsNotLimited(t *testing.T) {
	ss := NewSessions(time.Hour, false)
	g := &Guard{Sessions: ss, AnonLimiter: NewRateLimiter(0.01, 2)}
	h := NewHandler(hardeningDispatcher(t), nil, g)
	token, sess := ss.Create("u1", nil)
	for i := 0; i < 10; i++ {
		req := httptest.NewRequest(http.MethodPost, "/api/rpc/Add", strings.NewReader("[1,2]"))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set(csrfHeader, sess.CSRF)
		req.AddCookie(&http.Cookie{Name: cookieName, Value: token})
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("authenticated request %d: %d %s", i, rec.Code, rec.Body.String())
		}
	}
	// а анонимные по-прежнему упираются в лимит
	var limited bool
	for i := 0; i < 4; i++ {
		limited = limited || anonRPC(h, "").Code == http.StatusTooManyRequests
	}
	if !limited {
		t.Error("anonymous requests must still be limited")
	}
}
