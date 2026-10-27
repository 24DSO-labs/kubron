// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"testing"
	"time"
)

func recv(t *testing.T, c *Conn) map[string]any {
	t.Helper()
	select {
	case b := <-c.Out():
		var m map[string]any
		if err := json.Unmarshal(b, &m); err != nil {
			t.Fatalf("не JSON: %s", b)
		}
		return m
	case <-time.After(time.Second):
		t.Fatal("кадр не пришёл")
		return nil
	}
}

func noFrame(t *testing.T, c *Conn) {
	t.Helper()
	select {
	case b := <-c.Out():
		t.Fatalf("лишний кадр: %s", b)
	case <-time.After(30 * time.Millisecond):
	}
}

func TestHubDeliversOnlyToSubscribers(t *testing.T) {
	h := NewHub()
	a, b := h.Register("s1"), h.Register("s2")
	a.Subscribe([]string{"cluster:changed", "terminal:data:7"})
	b.Subscribe([]string{"other"})

	h.Emit("cluster:changed", "prod", 3)
	m := recv(t, a)
	if m["event"] != "cluster:changed" {
		t.Fatalf("кадр: %v", m)
	}
	if d := m["data"].([]any); len(d) != 2 || d[0] != "prod" || d[1] != float64(3) {
		t.Errorf("data: %v", d)
	}
	noFrame(t, b)

	a.Unsubscribe([]string{"cluster:changed"})
	h.Emit("cluster:changed", "x")
	noFrame(t, a)
}

func TestHubEmitWithoutDataSendsEmptyArray(t *testing.T) {
	h := NewHub()
	c := h.Register("s")
	c.Subscribe([]string{"terminal:exit:1"})
	h.Emit("terminal:exit:1")
	if d, ok := recv(t, c)["data"].([]any); !ok || len(d) != 0 {
		t.Errorf("data должен быть []: %v", d)
	}
}

func TestHubSlowConsumerIsDropped(t *testing.T) {
	h := NewHub()
	c := h.Register("s")
	c.Subscribe([]string{"e"})
	for i := 0; i < connBuffer+5; i++ {
		h.Emit("e", i)
	}
	select {
	case <-c.Done():
	case <-time.After(time.Second):
		t.Fatal("медленный потребитель не отключён")
	}
	if h.Len() != 0 {
		t.Errorf("соединение осталось в хабе: %d", h.Len())
	}
	h.Emit("e", 1) // не должно паниковать
}

func TestHubCloseSession(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.Register("a"), h.Register("a"), h.Register("b")
	h.CloseSession("a")
	for _, c := range []*Conn{a1, a2} {
		select {
		case <-c.Done():
		default:
			t.Error("соединение сессии a не закрыто")
		}
	}
	select {
	case <-b.Done():
		t.Error("чужое соединение закрыто")
	default:
	}
	a1.Close() // идемпотентно
	if h.Len() != 1 {
		t.Errorf("в хабе %d, ждали 1", h.Len())
	}
}

func TestConnProtocol(t *testing.T) {
	h := NewHub()
	c := h.Register("s")
	var m map[string]any
	_ = json.Unmarshal(c.Handle([]byte(`{"op":"sub","events":["a","b"]}`)), &m)
	if m["op"] != "sub_ok" {
		t.Errorf("sub: %v", m)
	}
	h.Emit("a", 1)
	recv(t, c)
	_ = json.Unmarshal(c.Handle([]byte(`{"op":"unsub","events":["a"]}`)), &m)
	if m["op"] != "unsub_ok" {
		t.Errorf("unsub: %v", m)
	}
	_ = json.Unmarshal(c.Handle([]byte(`{"op":"ping"}`)), &m)
	if m["op"] != "pong" {
		t.Errorf("ping: %v", m)
	}
	for _, bad := range []string{`не json`, `{"op":"drop"}`} {
		_ = json.Unmarshal(c.Handle([]byte(bad)), &m)
		if m["op"] != "error" {
			t.Errorf("%s: %v", bad, m)
		}
	}
}

func TestSubscriptionLimit(t *testing.T) {
	c := NewHub().Register("s")
	ev := make([]string, maxSubscriptions+1)
	for i := range ev {
		ev[i] = "e" + string(rune('a'+i%26)) + string(rune('0'+i/26%10)) + string(rune('A'+i/260))
	}
	var m map[string]any
	raw, _ := json.Marshal(map[string]any{"op": "sub", "events": ev})
	_ = json.Unmarshal(c.Handle(raw), &m)
	if m["op"] != "error" {
		t.Errorf("ждали ошибку предела: %v", m["op"])
	}
}

func TestSessionsOnEndNotified(t *testing.T) {
	ss := NewSessions(time.Hour, true)
	var ended []string
	ss.OnEnd = func(id string) { ended = append(ended, id) }
	t1, s1 := ss.Create("a", nil)
	_, s2 := ss.Create("b", nil) // вытесняет первую
	if len(ended) != 1 || ended[0] != s1.ID || s1.ID == s2.ID || s1.ID == "" {
		t.Fatalf("вытеснение: %v (%q, %q)", ended, s1.ID, s2.ID)
	}
	if _, ok := ss.Get(t1); ok {
		t.Error("вытесненная сессия жива")
	}
	t2, _ := ss.Create("c", nil)
	ended = nil
	ss.Delete(t2)
	ss.Delete(t2) // повторное удаление не уведомляет
	if len(ended) != 1 {
		t.Errorf("Delete: %v", ended)
	}
}

func TestHubEmitToReachesOnlyOwnerSession(t *testing.T) {
	h := NewHub()
	a1, a2, b := h.Register("A"), h.Register("A"), h.Register("B")
	for _, c := range []*Conn{a1, a2, b} {
		c.Subscribe([]string{"logs:data:1"}) // подписка на чужой id событий не даёт
	}
	h.EmitTo("A", "logs:data:1", "строка")
	if m := recv(t, a1); m["event"] != "logs:data:1" {
		t.Fatalf("кадр: %v", m)
	}
	recv(t, a2) // все соединения владельца
	noFrame(t, b)

	// Адресная доставка не обходит подписку.
	a1.Unsubscribe([]string{"logs:data:1"})
	h.EmitTo("A", "logs:data:1", "ещё")
	noFrame(t, a1)
}
