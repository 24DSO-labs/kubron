// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// viewFixture: у каждого входа свой guardTarget (вид ядра), как в боевом сервере.
type viewFixture struct {
	h        http.Handler
	def      *guardTarget
	ss       *Sessions
	mu       sync.Mutex
	created  []*guardTarget
	sources  []string
	released []*guardTarget
}

func newViewFixture(t *testing.T, exclusive bool) *viewFixture {
	t.Helper()
	f := &viewFixture{def: &guardTarget{}}
	d, err := NewDispatcher(f.def, []string{"Login", "Logout", "AuthStatus", "Add"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	f.ss = NewSessions(time.Hour, exclusive)
	f.ss.Release = func(v any) {
		f.mu.Lock()
		f.released = append(f.released, v.(*guardTarget))
		f.mu.Unlock()
	}
	g := &Guard{Sessions: f.ss, TrustProxy: true}
	g.NewView = func(src string) any {
		v := &guardTarget{}
		f.mu.Lock()
		f.created = append(f.created, v)
		f.sources = append(f.sources, src)
		f.mu.Unlock()
		return v
	}
	f.h = NewHandler(d, nil, g)
	return f
}

func TestSessionsHaveIndependentViews(t *testing.T) {
	f := newViewFixture(t, false)
	ca, csrfA := login(t, f.h, "ann")
	cb, csrfB := login(t, f.h, "bob")
	if len(f.created) != 2 {
		t.Fatalf("%d views, expected 2", len(f.created))
	}
	if f.created[0].user.Username != "ann" || f.created[1].user.Username != "bob" {
		t.Fatal("login must run on its own view")
	}
	if f.def.user != nil {
		t.Fatal("the shared instance must not get a user")
	}
	for name, c := range map[string]struct {
		ck   *http.Cookie
		csrf string
	}{"ann": {ca, csrfA}, "bob": {cb, csrfB}} {
		_, rep := call(f.h, "AuthStatus", `[]`, c.ck, c.csrf)
		var st struct {
			User *struct{ Username string } `json:"user"`
		}
		_ = json.Unmarshal(rep.Result, &st)
		if rep.Error != nil || st.User == nil || st.User.Username != name {
			t.Errorf("%s sees %s", name, rep.Result)
		}
	}
	// Выход одной сессии не затрагивает другую.
	call(f.h, "Logout", `[]`, ca, csrfA)
	if rec, _ := call(f.h, "Add", `[1,2]`, cb, csrfB); rec.Code != 200 {
		t.Errorf("bob's session must stay alive: %d", rec.Code)
	}
	if rec, _ := call(f.h, "Add", `[1,2]`, ca, csrfA); rec.Code != 401 {
		t.Errorf("ann's session is closed: %d", rec.Code)
	}
	if len(f.released) != 1 || f.released[0] != f.created[0] {
		t.Errorf("ann's view must be released: %v", f.released)
	}
}

func TestExclusiveEvictionReleasesView(t *testing.T) {
	f := newViewFixture(t, true)
	login(t, f.h, "ann")
	login(t, f.h, "bob")
	if len(f.released) != 1 || f.released[0] != f.created[0] {
		t.Errorf("the evicted view was not released: %v", f.released)
	}
}

func TestFailedLoginReleasesFreshView(t *testing.T) {
	f := newViewFixture(t, false)
	rec, _ := call(f.h, "Login", `["ann","bad"]`, nil, "")
	if rec.Code == 200 {
		t.Fatal("login with a wrong password must not succeed")
	}
	if len(f.created) != 1 || len(f.released) != 1 || f.released[0] != f.created[0] {
		t.Errorf("%d created, %d released", len(f.created), len(f.released))
	}
	if f.ss.Len() != 0 {
		t.Error("there must be no session")
	}
}

func TestLoginSourceUsesRemoteAddrOrTrustedProxy(t *testing.T) {
	f := newViewFixture(t, false)
	req := httptest.NewRequest(http.MethodPost, "/api/rpc/Login", nil)
	req.RemoteAddr = "10.1.2.3:5555"
	if got := clientIP(req, false); got != "10.1.2.3" {
		t.Errorf("RemoteAddr: %q", got)
	}
	req.Header.Set("X-Forwarded-For", "6.6.6.6, 203.0.113.9")
	if got := clientIP(req, true); got != "203.0.113.9" {
		t.Errorf("trusted proxy: %q", got)
	}
	if got := clientIP(req, false); got != "10.1.2.3" {
		t.Errorf("XFF is ignored without trust: %q", got)
	}
	login(t, f.h, "ann") // httptest: RemoteAddr 192.0.2.1:1234
	if f.sources[0] != "192.0.2.1" {
		t.Errorf("login source: %q", f.sources[0])
	}
}

func TestCallOnTargetValidation(t *testing.T) {
	d, err := NewDispatcher(&guardTarget{}, []string{"Add"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	own := &guardTarget{user: nil}
	if _, err := d.CallOn(own, "Add", raw("1", "2")); err == nil {
		t.Error("an error was expected without a user on its own view")
	}
	if _, err := d.CallOn(&fakeTarget{}, "Add", raw("1", "2")); err == nil {
		t.Error("a target of a foreign type must be rejected")
	}
	if _, err := d.CallOn(nil, "Add", raw("1", "2")); err == nil {
		t.Error("nil is the shared instance (no user): a login error was expected")
	}
}
