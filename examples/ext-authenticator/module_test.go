package example

import (
	"context"
	"testing"

	"analog/pkg/ext"
)

func TestExternalModuleRegistersAuthenticator(t *testing.T) {
	r := ext.NewRegistry()
	if err := r.Install(New(map[string]string{"alice": "pw"})); err != nil {
		t.Fatal(err)
	}
	a, ok := r.Authenticator("static")
	if !ok {
		t.Fatal("аутентификатор не зарегистрирован")
	}
	pa, ok := a.(ext.PasswordAuthenticator)
	if !ok {
		t.Fatal("ждали PasswordAuthenticator")
	}
	id, err := pa.Authenticate(context.Background(), "alice", "pw")
	if err != nil || id.Username != "alice" {
		t.Fatalf("вход: %+v %v", id, err)
	}
	if _, err := pa.Authenticate(context.Background(), "alice", "wrong"); err == nil {
		t.Fatal("неверный пароль должен отклоняться")
	}
}
