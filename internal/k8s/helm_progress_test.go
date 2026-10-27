// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
)

func TestBeginHelmOpValidation(t *testing.T) {
	m := NewManager()
	if _, tr, err := m.beginHelmOp(context.Background(), "", "install", "prod", "web", true); err != nil || tr != nil {
		t.Errorf("пустой opID — операция без отслеживания: %v %v", tr, err)
	}
	for _, bad := range []string{"short", "has space 123456", "../../etc/passwd", strings.Repeat("a", 65)} {
		if _, _, err := m.beginHelmOp(context.Background(), bad, "install", "prod", "web", true); err == nil {
			t.Errorf("%q должен отклоняться", bad)
		}
	}
}

func TestHelmOpLifecycleAndCancel(t *testing.T) {
	m := NewManager()
	ctx, tr, err := m.beginHelmOp(context.Background(), "op-12345678", "install", "prod", "web", true)
	if err != nil || tr == nil {
		t.Fatal(err)
	}
	if _, _, err := m.beginHelmOp(context.Background(), "op-12345678", "install", "prod", "web", true); err == nil {
		t.Error("повторный opID должен отклоняться, пока операция идёт")
	}
	if err := m.CancelHelmOperation("op-12345678"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Error("отмена должна отменять контекст операции")
	}
	tr.finish(context.Canceled)
	tr.finish(nil) // повторный вызов безопасен
	if err := m.CancelHelmOperation("op-12345678"); err == nil {
		t.Error("завершённая операция не должна находиться")
	}
}

func TestHelmOpNotCancelable(t *testing.T) {
	m := NewManager()
	_, tr, err := m.beginHelmOp(context.Background(), "op-87654321", "rollback", "prod", "web", false)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.finish(nil)
	if err := m.CancelHelmOperation("op-87654321"); err == nil {
		t.Error("rollback отменять нельзя")
	}
}

func TestNilTrackerIsSafe(t *testing.T) {
	var tr *helmTracker
	tr.setPhase("x")
	tr.finish(errors.New("e"))
	if tr.handler() != nil {
		t.Error("nil-трекер не даёт handler")
	}
}

func TestProgressHandlerCollectsLines(t *testing.T) {
	m := NewManager()
	_, tr, err := m.beginHelmOp(context.Background(), "op-abcdef12", "upgrade", "prod", "web", true)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.finish(nil)

	log := slog.New(tr.handler()).With("release", "web")
	log.Debug("waiting for resources", "count", 7)
	log.Info(strings.Repeat("я", helmProgressLineMx+50))

	p := tr.snapshot(false, "")
	if len(p.Lines) != 2 {
		t.Fatalf("строк: %d (%v)", len(p.Lines), p.Lines)
	}
	if p.Lines[0] != "waiting for resources release=web count=7" {
		t.Errorf("формат строки: %q", p.Lines[0])
	}
	if got := len([]rune(p.Lines[1])); got != helmProgressLineMx+1 { // + «…»
		t.Errorf("длинная строка должна обрезаться, получено %d рун", got)
	}
	if p.Op != "upgrade" || p.Release != "web" || !p.Cancelable {
		t.Errorf("snapshot: %+v", p)
	}
}

func TestProgressBatchLimit(t *testing.T) {
	m := NewManager()
	_, tr, _ := m.beginHelmOp(context.Background(), "op-batch123", "install", "prod", "web", false)
	defer tr.finish(nil)
	for i := 0; i < helmProgressBatch+10; i++ {
		tr.addLine("line")
	}
	if p := tr.snapshot(false, ""); len(p.Lines) != helmProgressBatch {
		t.Errorf("пачка должна быть ограничена: %d", len(p.Lines))
	}
	if p := tr.snapshot(false, ""); len(p.Lines) != 10 {
		t.Errorf("остаток во второй пачке: %d", len(p.Lines))
	}
}
