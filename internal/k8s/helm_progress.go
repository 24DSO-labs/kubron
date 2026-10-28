// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"regexp"
	"strings"
	"sync"
	"time"
)

// Прогресс длинных Helm-операций (H6).
//
// Идентификатор операции (opID) создаёт фронтенд до вызова, поэтому события можно привязать к операции
// ещё до того, как блокирующий вызов вернётся. Бэкенд шлёт событие helm:progress:
//   - при смене фазы («Загрузка чарта», «Применение», «Ожидание готовности»);
//   - пачкой строк журнала Helm (его slog-сообщения: «waiting for resources count=7» и т. п.), не чаще
//     helmProgressTick, чтобы не засыпать UI;
//   - раз в секунду как heartbeat с прошедшим временем;
//   - итоговым событием done=true (с error при неудаче).
const (
	EventHelmProgress = "helm:progress"

	helmProgressTick   = 250 * time.Millisecond
	helmProgressBeat   = time.Second
	helmProgressBatch  = 40  // строк журнала в одном событии
	helmProgressLineMx = 300 // символов в строке
)

var helmOpIDRe = regexp.MustCompile(`^[A-Za-z0-9-]{8,64}$`)

// HelmProgress — полезная нагрузка события helm:progress.
type HelmProgress struct {
	OpID       string   `json:"opId"`
	Op         string   `json:"op"` // install | upgrade | rollback | uninstall
	Namespace  string   `json:"namespace"`
	Release    string   `json:"release"`
	Phase      string   `json:"phase"`
	Lines      []string `json:"lines"` // новые строки журнала Helm
	ElapsedMs  int64    `json:"elapsedMs"`
	Cancelable bool     `json:"cancelable"`
	Done       bool     `json:"done"`
	Error      string   `json:"error"`
}

type helmOps struct {
	mu  sync.Mutex
	ops map[string]*helmTracker
}

var activeHelmOps = &helmOps{ops: map[string]*helmTracker{}}

// helmTracker отслеживает одну операцию. Методы безопасны для nil-получателя (операция без opID).
type helmTracker struct {
	m       *Manager
	opID    string
	op      string
	ns, rel string
	started time.Time
	cancel  context.CancelFunc // nil — операцию нельзя отменить

	mu      sync.Mutex
	phase   string
	pending []string
	done    bool
	stop    chan struct{}
}

// beginHelmOp регистрирует операцию. Пустой opID — операция не отслеживается (возвращается nil-трекер).
// cancelable: операция принимает контекст, и CancelHelmOperation её прервёт.
func (m *Manager) beginHelmOp(ctx context.Context, opID, op, ns, rel string, cancelable bool) (context.Context, *helmTracker, error) {
	if opID == "" {
		return ctx, nil, nil
	}
	if !helmOpIDRe.MatchString(opID) {
		return ctx, nil, errors.New("некорректный идентификатор операции")
	}
	t := &helmTracker{m: m, opID: opID, op: op, ns: ns, rel: rel, started: time.Now(), stop: make(chan struct{})}
	if cancelable {
		ctx, t.cancel = context.WithCancel(ctx)
	}

	activeHelmOps.mu.Lock()
	if _, busy := activeHelmOps.ops[opID]; busy {
		activeHelmOps.mu.Unlock()
		if t.cancel != nil {
			t.cancel()
		}
		return ctx, nil, errors.New("операция с таким идентификатором уже выполняется")
	}
	activeHelmOps.ops[opID] = t
	activeHelmOps.mu.Unlock()

	go t.loop()
	return ctx, t, nil
}

// CancelHelmOperation прерывает операцию, поддерживающую отмену (установка и апгрейд).
func (m *Manager) CancelHelmOperation(opID string) error {
	activeHelmOps.mu.Lock()
	t := activeHelmOps.ops[opID]
	activeHelmOps.mu.Unlock()
	if t == nil {
		return errors.New("операция не найдена или уже завершена")
	}
	if t.cancel == nil {
		return errors.New("эту операцию нельзя отменить")
	}
	t.setPhase("Отмена…")
	t.cancel()
	return nil
}

func (t *helmTracker) snapshot(done bool, errMsg string) HelmProgress {
	t.mu.Lock()
	defer t.mu.Unlock()
	n := len(t.pending)
	if n > helmProgressBatch {
		n = helmProgressBatch
	}
	lines := append([]string(nil), t.pending[:n]...)
	t.pending = t.pending[n:]
	return HelmProgress{
		OpID: t.opID, Op: t.op, Namespace: t.ns, Release: t.rel, Phase: t.phase, Lines: lines,
		ElapsedMs: time.Since(t.started).Milliseconds(), Cancelable: t.cancel != nil, Done: done, Error: errMsg,
	}
}

func (t *helmTracker) send(p HelmProgress) {
	if t.m != nil {
		t.m.emit(EventHelmProgress, p)
	}
}

// loop периодически сбрасывает накопленные строки и шлёт heartbeat.
func (t *helmTracker) loop() {
	tick := time.NewTicker(helmProgressTick)
	defer tick.Stop()
	lastBeat := time.Now()
	for {
		select {
		case <-t.stop:
			return
		case <-tick.C:
			t.mu.Lock()
			has := len(t.pending) > 0
			t.mu.Unlock()
			if has || time.Since(lastBeat) >= helmProgressBeat {
				t.send(t.snapshot(false, ""))
				lastBeat = time.Now()
			}
		}
	}
}

func (t *helmTracker) setPhase(phase string) {
	if t == nil {
		return
	}
	t.mu.Lock()
	t.phase = phase
	t.mu.Unlock()
	t.send(t.snapshot(false, ""))
}

// finish завершает операцию: итоговое событие и снятие с учёта. Вызывать ровно один раз (defer).
func (t *helmTracker) finish(err error) {
	if t == nil {
		return
	}
	t.mu.Lock()
	if t.done {
		t.mu.Unlock()
		return
	}
	t.done = true
	t.mu.Unlock()
	close(t.stop)

	activeHelmOps.mu.Lock()
	delete(activeHelmOps.ops, t.opID)
	activeHelmOps.mu.Unlock()
	if t.cancel != nil {
		t.cancel()
	}

	msg := ""
	if err != nil {
		msg = err.Error()
		if errors.Is(err, context.Canceled) {
			msg = "операция отменена"
		}
	}
	for {
		p := t.snapshot(false, "")
		if len(p.Lines) == 0 {
			break
		}
		t.send(p)
	}
	t.send(t.snapshot(true, msg))
}

// addLine кладёт строку журнала в очередь (с обрезкой и защитой от разрастания).
func (t *helmTracker) addLine(s string) {
	s = strings.TrimSpace(s)
	if s == "" {
		return
	}
	if len([]rune(s)) > helmProgressLineMx {
		s = string([]rune(s)[:helmProgressLineMx]) + "…"
	}
	t.mu.Lock()
	if len(t.pending) < 1000 {
		t.pending = append(t.pending, s)
	}
	t.mu.Unlock()
}

// handler возвращает slog.Handler для action.Configuration: журнал Helm превращается в строки прогресса.
// nil-трекер даёт nil — тогда Helm пользуется логгером по умолчанию.
func (t *helmTracker) handler() slog.Handler {
	if t == nil {
		return nil
	}
	return &helmProgressHandler{t: t}
}

type helmProgressHandler struct {
	t     *helmTracker
	attrs []slog.Attr
}

func (h *helmProgressHandler) Enabled(context.Context, slog.Level) bool { return true }

func (h *helmProgressHandler) Handle(_ context.Context, r slog.Record) error {
	var b strings.Builder
	b.WriteString(r.Message)
	write := func(a slog.Attr) bool {
		b.WriteString(" ")
		b.WriteString(a.Key)
		b.WriteString("=")
		fmt.Fprint(&b, a.Value.Resolve().Any())
		return true
	}
	for _, a := range h.attrs {
		write(a)
	}
	r.Attrs(write)
	h.t.addLine(b.String())
	return nil
}

func (h *helmProgressHandler) WithAttrs(attrs []slog.Attr) slog.Handler {
	return &helmProgressHandler{t: h.t, attrs: append(append([]slog.Attr(nil), h.attrs...), attrs...)}
}

func (h *helmProgressHandler) WithGroup(string) slog.Handler { return h }
