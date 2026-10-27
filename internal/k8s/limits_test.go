// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import "testing"

func TestDefaultLimitsAndInheritance(t *testing.T) {
	m := NewManager()
	if got := m.Limits(); got.LogStreams != logMaxStreams || got.Terminals != 0 || got.PortForwards != 0 {
		t.Fatalf("лимиты десктопа по умолчанию: %+v", got)
	}
	m.SetLimits(DefaultLimits())
	v := m.ForSession()
	if v.Limits() != (Limits{Terminals: 8, LogStreams: 20, PortForwards: 10}) {
		t.Fatalf("вид не унаследовал лимиты: %+v", v.Limits())
	}
	// Виды независимы: терминалы и логи у каждого свои.
	if v.terminalSessions == m.terminalSessions {
		t.Error("у вида должен быть свой менеджер терминалов")
	}
}

func TestLogStreamsLimitIsPerView(t *testing.T) {
	var a, b logStreams
	noop := func() *logEntry { return &logEntry{cancel: func() {}} }
	for _, id := range []string{"1", "2"} {
		if !a.add(id, noop(), 2) {
			t.Fatalf("поток %s должен приниматься", id)
		}
	}
	if a.add("3", noop(), 2) {
		t.Error("третий поток сверх лимита 2 должен отклоняться")
	}
	if !a.add("1", noop(), 2) {
		t.Error("замена потока с тем же id лимит не занимает")
	}
	if !b.add("x", noop(), 2) {
		t.Error("лимит одного вида не должен влиять на другой")
	}
	if !a.add("4", noop(), 0) {
		t.Error("0 — без ограничения")
	}
}

func TestTerminalTryAddLimit(t *testing.T) {
	tsm := NewTerminalSessionManager()
	if !tsm.tryAdd(&TerminalSession{ID: "a"}, 2) || !tsm.tryAdd(&TerminalSession{ID: "b"}, 2) {
		t.Fatal("первые два терминала должны приниматься")
	}
	if tsm.tryAdd(&TerminalSession{ID: "c"}, 2) {
		t.Error("третий терминал сверх лимита")
	}
	if !tsm.tryAdd(&TerminalSession{ID: "c"}, 0) || tsm.count() != 3 {
		t.Error("0 — без ограничения")
	}
	var nilTSM *TerminalSessionManager
	if nilTSM.count() != 0 {
		t.Error("count nil-менеджера")
	}
}

func TestValidStreamID(t *testing.T) {
	for id, want := range map[string]bool{
		"log_6f1c2d3e-aaaa-bbbb-cccc-0123456789ab": true,
		"": false, "a b": false, "logs:data:1": false, "../x": false,
		"line\nbreak": false,
	} {
		if validStreamID(id) != want {
			t.Errorf("validStreamID(%q) != %v", id, want)
		}
	}
	long := make([]byte, 65)
	for i := range long {
		long[i] = 'a'
	}
	if validStreamID(string(long)) {
		t.Error("id длиннее 64 символов должен отклоняться")
	}
}
