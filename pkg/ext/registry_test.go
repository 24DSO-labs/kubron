// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package ext

import (
	"context"
	"errors"
	"net/url"
	"strings"
	"testing"
)

type fakeModule struct {
	name    string
	version int
	reg     func(*Registry) error
}

func (m fakeModule) Name() string               { return m.name }
func (m fakeModule) APIVersion() int            { return m.version }
func (m fakeModule) Register(r *Registry) error { return m.reg(r) }

type pwAuth struct{ id string }

func (a pwAuth) ID() string { return a.id }
func (a pwAuth) Kind() Kind { return KindPassword }
func (a pwAuth) Authenticate(context.Context, string, string) (*Identity, error) {
	return &Identity{Subject: "u1", Username: "alice"}, nil
}

type redirAuth struct{ id string }

func (a redirAuth) ID() string { return a.id }
func (a redirAuth) Kind() Kind { return KindRedirect }
func (a redirAuth) BeginLogin(context.Context, string) (*RedirectStart, error) {
	return &RedirectStart{URL: "https://idp.example/auth", State: []byte("s")}, nil
}
func (a redirAuth) CompleteLogin(_ context.Context, state []byte, p url.Values) (*Identity, error) {
	if string(state) != "s" || p.Get("code") == "" {
		return nil, errors.New("bad callback")
	}
	return &Identity{Subject: "u2", Username: "bob", Groups: []string{"devops"}}, nil
}

// lying объявляет kind=redirect, но не реализует RedirectAuthenticator.
type lying struct{}

func (lying) ID() string { return "liar" }
func (lying) Kind() Kind { return KindRedirect }

func TestInstallRegistersAuthenticators(t *testing.T) {
	r := NewRegistry()
	err := r.Install(fakeModule{"sso", APIVersion, func(r *Registry) error {
		if err := r.RegisterAuthenticator(pwAuth{"local"}); err != nil {
			return err
		}
		return r.RegisterAuthenticator(redirAuth{"oidc"})
	}})
	if err != nil {
		t.Fatal(err)
	}
	if got := r.Modules(); len(got) != 1 || got[0] != "sso" {
		t.Fatalf("Modules = %v", got)
	}
	ids := []string{}
	for _, a := range r.Authenticators() {
		ids = append(ids, a.ID())
	}
	if strings.Join(ids, ",") != "local,oidc" {
		t.Fatalf("порядок регистрации потерян: %v", ids)
	}
	a, ok := r.Authenticator("oidc")
	if !ok {
		t.Fatal("oidc не найден")
	}
	start, err := a.(RedirectAuthenticator).BeginLogin(context.Background(), "https://kubron/cb")
	if err != nil {
		t.Fatal(err)
	}
	id, err := a.(RedirectAuthenticator).CompleteLogin(context.Background(), start.State, url.Values{"code": {"x"}})
	if err != nil || id.Username != "bob" {
		t.Fatalf("CompleteLogin: %+v %v", id, err)
	}
}

func TestInstallRejectsWrongAPIVersion(t *testing.T) {
	r := NewRegistry()
	err := r.Install(fakeModule{"old", APIVersion + 1, func(*Registry) error {
		t.Fatal("Register не должен вызываться для чужой версии")
		return nil
	}})
	if err == nil || !strings.Contains(err.Error(), "old") {
		t.Fatalf("ждали ошибку версии с именем модуля, получили %v", err)
	}
	if len(r.Modules()) != 0 {
		t.Fatal("отвергнутый модуль не должен попадать в список")
	}
}

func TestRegisterAuthenticatorValidation(t *testing.T) {
	r := NewRegistry()
	if err := r.RegisterAuthenticator(nil); err == nil {
		t.Error("nil должен отклоняться")
	}
	if err := r.RegisterAuthenticator(pwAuth{""}); err == nil {
		t.Error("пустой ID должен отклоняться")
	}
	if err := r.RegisterAuthenticator(lying{}); err == nil {
		t.Error("kind без реализации интерфейса должен отклоняться")
	}
	if err := r.RegisterAuthenticator(pwAuth{"local"}); err != nil {
		t.Fatal(err)
	}
	if err := r.RegisterAuthenticator(pwAuth{"local"}); err == nil {
		t.Error("дубликат ID должен отклоняться")
	}
}

func TestInstallPropagatesModuleError(t *testing.T) {
	r := NewRegistry()
	boom := errors.New("boom")
	err := r.Install(fakeModule{"bad", APIVersion, func(*Registry) error { return boom }})
	if !errors.Is(err, boom) {
		t.Fatalf("ошибка модуля должна оборачиваться: %v", err)
	}
	if err := r.Install(nil); err == nil {
		t.Error("nil-модуль должен отклоняться")
	}
}
