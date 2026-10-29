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
		t.Errorf("an empty opID means an untracked operation: %v %v", tr, err)
	}
	for _, bad := range []string{"short", "has space 123456", "../../etc/passwd", strings.Repeat("a", 65)} {
		if _, _, err := m.beginHelmOp(context.Background(), bad, "install", "prod", "web", true); err == nil {
			t.Errorf("%q must be rejected", bad)
		}
	}
}

func TestHelmOpLifecycleAndCancel(t *testing.T) {
	setupHelmHome(t) // finish() пишет журнал операций: изолируем от настоящего ~/.kubron
	m := NewManager()
	ctx, tr, err := m.beginHelmOp(context.Background(), "op-12345678", "install", "prod", "web", true)
	if err != nil || tr == nil {
		t.Fatal(err)
	}
	if _, _, err := m.beginHelmOp(context.Background(), "op-12345678", "install", "prod", "web", true); err == nil {
		t.Error("a duplicate opID must be rejected while the operation is running")
	}
	if err := m.CancelHelmOperation("op-12345678"); err != nil {
		t.Fatal(err)
	}
	if !errors.Is(ctx.Err(), context.Canceled) {
		t.Error("cancel must cancel the operation context")
	}
	tr.finish(context.Canceled)
	tr.finish(nil) // повторный вызов безопасен
	if err := m.CancelHelmOperation("op-12345678"); err == nil {
		t.Error("a finished operation must not be found")
	}
}

func TestHelmOpNotCancelable(t *testing.T) {
	setupHelmHome(t) // finish() пишет журнал операций: изолируем от настоящего ~/.kubron
	m := NewManager()
	_, tr, err := m.beginHelmOp(context.Background(), "op-87654321", "rollback", "prod", "web", false)
	if err != nil {
		t.Fatal(err)
	}
	defer tr.finish(nil)
	if err := m.CancelHelmOperation("op-87654321"); err == nil {
		t.Error("rollback cannot be canceled")
	}
}

func TestNilTrackerIsSafe(t *testing.T) {
	var tr *helmTracker
	tr.setPhase("x")
	tr.finish(errors.New("e"))
	if tr.handler() != nil {
		t.Error("a nil tracker gives no handler")
	}
}

func TestProgressHandlerCollectsLines(t *testing.T) {
	setupHelmHome(t) // finish() пишет журнал операций: изолируем от настоящего ~/.kubron
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
		t.Fatalf("lines: %d (%v)", len(p.Lines), p.Lines)
	}
	if p.Lines[0] != "waiting for resources release=web count=7" {
		t.Errorf("line format: %q", p.Lines[0])
	}
	if got := len([]rune(p.Lines[1])); got != helmProgressLineMx+1 { // + «…»
		t.Errorf("a long line must be truncated, got %d runes", got)
	}
	if p.Op != "upgrade" || p.Release != "web" || !p.Cancelable {
		t.Errorf("snapshot: %+v", p)
	}
}

func TestProgressBatchLimit(t *testing.T) {
	setupHelmHome(t) // finish() пишет журнал операций: изолируем от настоящего ~/.kubron
	m := NewManager()
	_, tr, _ := m.beginHelmOp(context.Background(), "op-batch123", "install", "prod", "web", false)
	defer tr.finish(nil)
	for i := 0; i < helmProgressBatch+10; i++ {
		tr.addLine("line")
	}
	if p := tr.snapshot(false, ""); len(p.Lines) != helmProgressBatch {
		t.Errorf("the batch must be limited: %d", len(p.Lines))
	}
	if p := tr.snapshot(false, ""); len(p.Lines) != 10 {
		t.Errorf("the remainder in the second batch: %d", len(p.Lines))
	}
}
