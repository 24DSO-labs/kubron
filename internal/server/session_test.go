// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"analog/internal/auth"
)

// guardTarget имитирует ядро с одной общей сессией: Login запоминает пользователя.
type guardTarget struct{ user *auth.UserInfo }

func (g *guardTarget) Login(name, pass string) (*auth.UserInfo, error) {
	if pass != "ok" {
		return nil, auth.ErrInvalidCredentials
	}
	g.user = &auth.UserInfo{ID: "id-" + name, Username: name}
	return g.user, nil
}
func (g *guardTarget) Logout() { g.user = nil }
func (g *guardTarget) AuthStatus() (*auth.Status, error) {
	return &auth.Status{User: g.user}, nil
}
func (g *guardTarget) Add(a, b int) (int, error) {
	if g.user == nil {
		return 0, auth.ErrNotAuthenticated
	}
	return a + b, nil
}

func newGuarded(t *testing.T) (http.Handler, *guardTarget, *Sessions) {
	t.Helper()
	tg := &guardTarget{}
	d, err := NewDispatcher(tg, []string{"Login", "Logout", "AuthStatus", "Add"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	ss := NewSessions(time.Hour, true)
	return NewHandler(d, nil, &Guard{Sessions: ss}), tg, ss
}

func cookieOf(rec *httptest.ResponseRecorder) *http.Cookie {
	for _, c := range rec.Result().Cookies() {
		if c.Name == cookieName || c.Name == secureCookieName {
			return c
		}
	}
	return nil
}

func call(h http.Handler, method, body string, c *http.Cookie, csrf string) (*httptest.ResponseRecorder, rpcReply) {
	req := httptest.NewRequest(http.MethodPost, "/api/rpc/"+method, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	if c != nil {
		req.AddCookie(c)
	}
	if csrf != "" {
		req.Header.Set(csrfHeader, csrf)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	var rep rpcReply
	_ = json.Unmarshal(rec.Body.Bytes(), &rep)
	return rec, rep
}

func login(t *testing.T, h http.Handler, name string) (*http.Cookie, string) {
	t.Helper()
	rec, rep := call(h, "Login", `["`+name+`","ok"]`, nil, "")
	c := cookieOf(rec)
	if rec.Code != 200 || rep.Error != nil || c == nil {
		t.Fatalf("login: %d %s", rec.Code, rec.Body.String())
	}
	return c, rec.Header().Get(csrfHeader)
}

func TestLoginSetsHardenedCookieAndCSRF(t *testing.T) {
	h, _, _ := newGuarded(t)
	c, csrf := login(t, h, "ann")
	if !c.HttpOnly || c.SameSite != http.SameSiteLaxMode || c.Path != "/" || c.Secure || c.MaxAge <= 0 {
		t.Errorf("cookie: %+v", c)
	}
	if len(c.Value) < 40 || csrf == "" || csrf == c.Value {
		t.Errorf("токены: cookie=%q csrf=%q", c.Value, csrf)
	}
}

func TestSecureCookieBehindTrustedProxyAndTLS(t *testing.T) {
	d, _ := NewDispatcher(&guardTarget{}, []string{"Login"}, nil)
	h := NewHandler(d, nil, &Guard{Sessions: NewSessions(time.Hour, true), TrustProxy: true})
	rec, _ := post(h, "/api/rpc/Login", `["a","ok"]`, map[string]string{"X-Forwarded-Proto": "https"})
	c := cookieOf(rec)
	if c == nil || !c.Secure || c.Name != secureCookieName {
		t.Errorf("ожидали __Host- cookie с Secure: %+v", c)
	}
	// Без доверия к прокси заголовок игнорируется.
	h = NewHandler(d, nil, &Guard{Sessions: NewSessions(time.Hour, true)})
	rec, _ = post(h, "/api/rpc/Login", `["a","ok"]`, map[string]string{"X-Forwarded-Proto": "https"})
	if c := cookieOf(rec); c == nil || c.Secure {
		t.Errorf("X-Forwarded-Proto без trust-proxy не должен включать Secure: %+v", c)
	}
}

func TestProtectedMethodNeedsSessionAndCSRF(t *testing.T) {
	h, _, _ := newGuarded(t)
	if rec, rep := call(h, "Add", `[1,2]`, nil, ""); rec.Code != 401 || rep.Error.Code != "unauthenticated" {
		t.Errorf("без сессии: %d %s", rec.Code, rec.Body.String())
	}
	c, csrf := login(t, h, "ann")
	if rec, rep := call(h, "Add", `[1,2]`, c, ""); rec.Code != 403 || rep.Error.Code != "csrf" {
		t.Errorf("без CSRF: %d %s", rec.Code, rec.Body.String())
	}
	if rec, _ := call(h, "Add", `[1,2]`, c, "чужой"); rec.Code != 403 {
		t.Errorf("с чужим CSRF: %d", rec.Code)
	}
	if rec, rep := call(h, "Add", `[1,2]`, c, csrf); rec.Code != 200 || string(rep.Result) != "3" {
		t.Errorf("с сессией и CSRF: %d %s", rec.Code, rec.Body.String())
	}
	forged := &http.Cookie{Name: cookieName, Value: "forged-token"}
	if rec, _ := call(h, "Add", `[1,2]`, forged, csrf); rec.Code != 401 {
		t.Errorf("чужой cookie: %d", rec.Code)
	}
}

func TestAuthStatusHidesUserWithoutSession(t *testing.T) {
	h, _, _ := newGuarded(t)
	c, csrf := login(t, h, "ann")
	_, anon := call(h, "AuthStatus", `[]`, nil, "")
	if strings.Contains(string(anon.Result), "ann") {
		t.Errorf("аноним видит пользователя: %s", anon.Result)
	}
	_, mine := call(h, "AuthStatus", `[]`, c, csrf)
	if !strings.Contains(string(mine.Result), "ann") {
		t.Errorf("владелец сессии не видит себя: %s", mine.Result)
	}
}

func TestExclusiveLoginRevokesPreviousSession(t *testing.T) {
	h, _, ss := newGuarded(t)
	c1, csrf1 := login(t, h, "ann")
	c2, csrf2 := login(t, h, "bob")
	if ss.Len() != 1 {
		t.Fatalf("сессий: %d, ждали 1", ss.Len())
	}
	if rec, _ := call(h, "Add", `[1,1]`, c1, csrf1); rec.Code != 401 {
		t.Errorf("старая сессия должна быть отозвана: %d", rec.Code)
	}
	if rec, _ := call(h, "Add", `[1,1]`, c2, csrf2); rec.Code != 200 {
		t.Errorf("новая сессия: %d", rec.Code)
	}
}

func TestFailedLoginKeepsSessionsAndSetsNoCookie(t *testing.T) {
	h, _, ss := newGuarded(t)
	c, csrf := login(t, h, "ann")
	rec, _ := call(h, "Login", `["bob","плохой"]`, nil, "")
	if rec.Code == 200 || cookieOf(rec) != nil {
		t.Errorf("неудачный вход: %d cookie=%v", rec.Code, cookieOf(rec))
	}
	if rec, _ := call(h, "Add", `[1,1]`, c, csrf); rec.Code != 200 || ss.Len() != 1 {
		t.Errorf("сессия ann должна жить: %d", rec.Code)
	}
}

func TestLogoutClosesSession(t *testing.T) {
	h, _, ss := newGuarded(t)
	c, csrf := login(t, h, "ann")
	rec, _ := call(h, "Logout", `[]`, c, csrf)
	if rec.Code != 200 || ss.Len() != 0 {
		t.Fatalf("logout: %d, сессий %d", rec.Code, ss.Len())
	}
	if cc := cookieOf(rec); cc == nil || cc.MaxAge >= 0 {
		t.Errorf("cookie должен удаляться: %+v", cc)
	}
	if rec, _ := call(h, "Add", `[1,1]`, c, csrf); rec.Code != 401 {
		t.Errorf("после выхода: %d", rec.Code)
	}
}

func TestCoreSessionLossDropsWebSession(t *testing.T) {
	h, tg, ss := newGuarded(t)
	c, csrf := login(t, h, "ann")
	tg.user = nil // ядро закрыло сессию по простою
	rec, _ := call(h, "Add", `[1,1]`, c, csrf)
	if rec.Code != 401 || ss.Len() != 0 {
		t.Errorf("ждали 401 и удаление сессии: %d, сессий %d", rec.Code, ss.Len())
	}
}

func TestSessionExpiresAbsolutely(t *testing.T) {
	ss := NewSessions(time.Minute, false)
	now := time.Now()
	ss.now = func() time.Time { return now }
	tok, _ := ss.Create("u")
	if _, ok := ss.Get(tok); !ok {
		t.Fatal("сессия должна быть жива")
	}
	now = now.Add(2 * time.Minute)
	if _, ok := ss.Get(tok); ok {
		t.Error("сессия должна истечь")
	}
}

func TestSessionsNonExclusiveKeepsMany(t *testing.T) {
	ss := NewSessions(time.Hour, false)
	ss.Create("a")
	ss.Create("b")
	if ss.Len() != 2 {
		t.Errorf("ждали 2, есть %d", ss.Len())
	}
}

func TestSessionEndpoint(t *testing.T) {
	h, _, _ := newGuarded(t)
	get := func(c *http.Cookie) map[string]any {
		req := httptest.NewRequest(http.MethodGet, "/api/session", nil)
		if c != nil {
			req.AddCookie(c)
		}
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		var m map[string]any
		_ = json.Unmarshal(rec.Body.Bytes(), &m)
		return m
	}
	if m := get(nil); m["authenticated"] != false || m["csrfToken"] != nil {
		t.Errorf("аноним: %v", m)
	}
	c, csrf := login(t, h, "ann")
	if m := get(c); m["authenticated"] != true || m["csrfToken"] != csrf {
		t.Errorf("со входом: %v", m)
	}
}

func TestCrossSiteFetchMetadataRejected(t *testing.T) {
	h, _, _ := newGuarded(t)
	for _, v := range []string{"cross-site", "same-site"} {
		if rec, _ := post(h, "/api/rpc/AuthStatus", `[]`, map[string]string{"Sec-Fetch-Site": v}); rec.Code != 403 {
			t.Errorf("Sec-Fetch-Site=%s: %d", v, rec.Code)
		}
	}
	if rec, _ := post(h, "/api/rpc/AuthStatus", `[]`, map[string]string{"Sec-Fetch-Site": "same-origin"}); rec.Code != 200 {
		t.Errorf("same-origin: %d", rec.Code)
	}
}
