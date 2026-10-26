// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"errors"
	"strings"
	"testing"
)

type fakeTarget struct{ lastBytes []byte }

type point struct {
	X int `json:"x"`
	Y int `json:"y"`
}

func (f *fakeTarget) Add(a, b int) (int, error)       { return a + b, nil }
func (f *fakeTarget) Fail() error                     { return errors.New("боль") }
func (f *fakeTarget) Nothing()                        {}
func (f *fakeTarget) Echo(p point, s string) point    { return point{p.X + len(s), p.Y} }
func (f *fakeTarget) Boom() error                     { panic("oops") }
func (f *fakeTarget) Write(id string, b []byte) error { f.lastBytes = b; return nil }
func (f *fakeTarget) Pair() (string, error)           { return "", errors.New("нет") }
func (f *fakeTarget) Variadic(xs ...int) int          { return len(xs) }
func (f *fakeTarget) TooMany() (int, int, error)      { return 0, 0, nil }
func (f *fakeTarget) hidden() int                     { return 1 }
func (f *fakeTarget) Secret() string                  { return "s" }

func raw(vals ...string) []json.RawMessage {
	out := make([]json.RawMessage, len(vals))
	for i, v := range vals {
		out[i] = json.RawMessage(v)
	}
	return out
}

func newTestDispatcher(t *testing.T) (*Dispatcher, *fakeTarget) {
	t.Helper()
	ft := &fakeTarget{}
	d, err := NewDispatcher(ft, []string{"Add", "Fail", "Nothing", "Echo", "Boom", "Write", "Pair"},
		map[string]string{"Secret": "закрыт"})
	if err != nil {
		t.Fatal(err)
	}
	return d, ft
}

func TestCallResultsAndErrors(t *testing.T) {
	d, _ := newTestDispatcher(t)

	if got, err := d.Call("Add", raw("2", "3")); err != nil || got != 5 {
		t.Fatalf("Add = %v, %v", got, err)
	}
	if _, err := d.Call("Fail", nil); err == nil || err.Error() != "боль" {
		t.Fatalf("Fail: %v", err)
	}
	if got, err := d.Call("Nothing", nil); err != nil || got != nil {
		t.Fatalf("Nothing = %v, %v", got, err)
	}
	if _, err := d.Call("Pair", nil); err == nil {
		t.Fatal("ошибка вторым результатом должна возвращаться")
	}
	got, err := d.Call("Echo", raw(`{"x":1,"y":2,"extra":true}`, `"abc"`))
	if err != nil || got != (point{4, 2}) {
		t.Fatalf("Echo = %v, %v (лишние поля JSON не должны ломать вызов)", got, err)
	}
}

func TestCallRejectsUnknownAndDenied(t *testing.T) {
	d, _ := newTestDispatcher(t)
	if _, err := d.Call("Nope", nil); !errors.Is(err, ErrUnknownMethod) {
		t.Errorf("неизвестный метод: %v", err)
	}
	if _, err := d.Call("Secret", nil); !errors.Is(err, ErrMethodDenied) {
		t.Errorf("запрещённый метод: %v", err)
	}
	// существует у объекта, но не в allowlist — недоступен
	if _, err := d.Call("TooMany", nil); !errors.Is(err, ErrUnknownMethod) {
		t.Errorf("метод вне allowlist: %v", err)
	}
	if _, err := d.Call("hidden", nil); !errors.Is(err, ErrUnknownMethod) {
		t.Errorf("неэкспортируемый метод: %v", err)
	}
}

func TestCallBadArgs(t *testing.T) {
	d, _ := newTestDispatcher(t)
	var bad *BadArgsError
	for name, c := range map[string]struct {
		method string
		args   []json.RawMessage
	}{
		"мало аргументов":  {"Add", raw("1")},
		"много аргументов": {"Add", raw("1", "2", "3")},
		"не тот тип":       {"Add", raw(`"x"`, "2")},
		"сломанный JSON":   {"Echo", raw(`{"x":`, `"s"`)},
	} {
		if _, err := d.Call(c.method, c.args); !errors.As(err, &bad) {
			t.Errorf("%s: ждали BadArgsError, получили %v", name, err)
		}
	}
}

func TestCallPanicBecomesError(t *testing.T) {
	d, _ := newTestDispatcher(t)
	_, err := d.Call("Boom", nil)
	if err == nil || !strings.Contains(err.Error(), "Boom") {
		t.Fatalf("паника должна стать ошибкой с именем метода: %v", err)
	}
	if strings.Contains(err.Error(), "oops") {
		t.Fatal("текст паники не должен утекать клиенту")
	}
}

func TestBytesArgAcceptsBase64AndNumberArray(t *testing.T) {
	d, ft := newTestDispatcher(t)
	if _, err := d.Call("Write", raw(`"s1"`, `"aGVsbG8="`)); err != nil || string(ft.lastBytes) != "hello" {
		t.Fatalf("base64: %v %q", err, ft.lastBytes)
	}
	if _, err := d.Call("Write", raw(`"s1"`, `[104,105]`)); err != nil || string(ft.lastBytes) != "hi" {
		t.Fatalf("массив чисел: %v %q", err, ft.lastBytes)
	}
	if _, err := d.Call("Write", raw(`"s1"`, `[300]`)); err == nil {
		t.Fatal("байт вне 0..255 должен отклоняться")
	}
	if _, err := d.Call("Write", raw(`"s1"`, `"!!!"`)); err == nil {
		t.Fatal("битый base64 должен отклоняться")
	}
}

func TestNewDispatcherValidatesAllowlist(t *testing.T) {
	ft := &fakeTarget{}
	for name, allowed := range map[string][]string{
		"нет метода":       {"Missing"},
		"variadic":         {"Variadic"},
		"три результата":   {"TooMany"},
		"неэкспортируемый": {"hidden"},
	} {
		if _, err := NewDispatcher(ft, allowed, nil); err == nil {
			t.Errorf("%s: ждали ошибку конфигурации", name)
		}
	}
	d, err := NewDispatcher(ft, []string{"Add", "Echo"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if m := d.Methods(); len(m) != 2 || m[0] != "Add" || m[1] != "Echo" {
		t.Fatalf("Methods = %v", m)
	}
}
