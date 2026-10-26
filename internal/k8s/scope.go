// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"regexp"
	"sort"
	"strings"
	"sync"
)

// Набор namespace для списков: пусто (или пустое имя в списке) = «все», 1 = один, N = набор.
// Kubernetes API не умеет list сразу по нескольким namespace, поэтому набор превращается
// в несколько запросов; весь связанный код (нормализация, лимиты, параллелизм, частичные отказы)
// собран здесь, а методы Manager только передают ему функцию одиночного запроса.

const (
	// maxScopeNamespaces — предел размера набора за один вызов.
	maxScopeNamespaces = 50
	// scopeConcurrency — сколько namespace читается одновременно.
	scopeConcurrency = 8
	// scopeListAllThreshold — при большем числе namespace (и праве на cluster-wide list)
	// один общий запрос с фильтрацией дешевле, чем N отдельных.
	scopeListAllThreshold = 5

	// EventListPartial — событие фронтенду: часть namespace прочитать не удалось.
	EventListPartial = "list:partial"
)

var nsNameRe = regexp.MustCompile(`^[a-z0-9]([-a-z0-9]*[a-z0-9])?$`)

// nsScope — нормализованный набор namespace. Пустой names означает «все».
type nsScope struct{ names []string }

func (s nsScope) all() bool { return len(s.names) == 0 }

// has сообщает, входит ли namespace в набор (для «все» — всегда).
func (s nsScope) has(ns string) bool {
	if s.all() {
		return true
	}
	for _, n := range s.names {
		if n == ns {
			return true
		}
	}
	return false
}

// parseScope проверяет и нормализует вход от фронтенда: убирает дубли, сортирует,
// валидирует имена по DNS-1123 и ограничивает размер. Пустая строка в списке означает «все».
// Имена попадают в PromQL и фильтры, поэтому доверять фронтенду нельзя.
func parseScope(in []string) (nsScope, error) {
	seen := make(map[string]bool, len(in))
	names := make([]string, 0, len(in))
	sawEmpty := false
	for _, raw := range in {
		n := strings.TrimSpace(raw)
		if n == "" {
			sawEmpty = true
			continue
		}
		if len(n) > 63 || !nsNameRe.MatchString(n) {
			return nsScope{}, fmt.Errorf("некорректное имя namespace: %q", clipName(n))
		}
		if !seen[n] {
			seen[n] = true
			names = append(names, n)
		}
	}
	if sawEmpty || len(names) == 0 {
		return nsScope{}, nil
	}
	if len(names) > maxScopeNamespaces {
		return nsScope{}, fmt.Errorf("выбрано слишком много namespace: %d (максимум %d)", len(names), maxScopeNamespaces)
	}
	sort.Strings(names)
	return nsScope{names: names}, nil
}

func clipName(s string) string {
	r := []rune(s)
	if len(r) > 64 {
		return string(r[:64]) + "…"
	}
	return s
}

// nsFailure — namespace, который не удалось прочитать.
type nsFailure struct {
	Namespace string `json:"namespace"`
	Error     string `json:"error"`
}

// partialListEvent — payload события EventListPartial.
type partialListEvent struct {
	Method   string      `json:"method"`
	Failures []nsFailure `json:"failures"`
}

// reportPartial сообщает фронтенду о namespace, которые не прочитались; без отказов ничего не делает.
func (m *Manager) reportPartial(method string, failures []nsFailure) {
	if len(failures) == 0 {
		return
	}
	m.emit(EventListPartial, partialListEvent{Method: method, Failures: failures})
}

// listScoped читает ресурсы набора namespace.
//
//   - «все» и один namespace — один запрос, ошибка возвращается как есть;
//   - несколько — при canListAll и N > scopeListAllThreshold один общий запрос с фильтрацией
//     (при его ошибке — запасной путь), иначе параллельные запросы с ограничением конкурентности;
//   - отказ части namespace не роняет вызов: они возвращаются в []nsFailure, ошибка только если
//     не прочитался ни один либо отменён ctx.
//
// Порядок результата детерминирован: по namespace; внутри namespace — как отдал API.
func listScoped[T any](
	ctx context.Context,
	scope nsScope,
	canListAll bool,
	list func(ctx context.Context, ns string) ([]T, error),
	nsOf func(T) string,
) ([]T, []nsFailure, error) {
	switch {
	case scope.all():
		items, err := list(ctx, "")
		return items, nil, err
	case len(scope.names) == 1:
		items, err := list(ctx, scope.names[0])
		return items, nil, err
	}

	if canListAll && len(scope.names) > scopeListAllThreshold {
		items, err := list(ctx, "")
		if err == nil {
			return filterByNamespace(items, scope.names, nsOf), nil, nil
		}
		if ctx.Err() != nil {
			return nil, nil, ctx.Err()
		}
		// нет права или сбой общего запроса: читаем по одному namespace
	}
	return listEach(ctx, scope.names, list)
}

func filterByNamespace[T any](items []T, names []string, nsOf func(T) string) []T {
	want := make(map[string]bool, len(names))
	for _, n := range names {
		want[n] = true
	}
	out := make([]T, 0, len(items))
	for _, it := range items {
		if want[nsOf(it)] {
			out = append(out, it)
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return nsOf(out[i]) < nsOf(out[j]) })
	return out
}

func listEach[T any](
	ctx context.Context,
	names []string,
	list func(ctx context.Context, ns string) ([]T, error),
) ([]T, []nsFailure, error) {
	type result struct {
		items []T
		err   error
	}
	results := make([]result, len(names))
	sem := make(chan struct{}, scopeConcurrency)
	var wg sync.WaitGroup
	for i, ns := range names {
		wg.Add(1)
		// i и ns передаются аргументами: не зависим от семантики переменной цикла (go < 1.22)
		go func(i int, ns string) {
			defer wg.Done()
			select {
			case sem <- struct{}{}:
			case <-ctx.Done():
				results[i].err = ctx.Err()
				return
			}
			defer func() { <-sem }()
			items, err := list(ctx, ns)
			results[i] = result{items: items, err: err}
		}(i, ns)
	}
	wg.Wait()

	if err := ctx.Err(); err != nil {
		return nil, nil, err
	}

	var (
		out      []T
		failures []nsFailure
		firstErr error
	)
	for i, r := range results {
		if r.err != nil {
			failures = append(failures, nsFailure{Namespace: names[i], Error: r.err.Error()})
			if firstErr == nil {
				firstErr = fmt.Errorf("%s: %w", names[i], r.err)
			}
			continue
		}
		out = append(out, r.items...)
	}
	if len(failures) == len(names) {
		return nil, failures, firstErr
	}
	if out == nil {
		out = []T{}
	}
	return out, failures, nil
}

// promNamespaceMatcher — селектор PromQL по набору namespace: точное совпадение для одного,
// регулярное выражение для нескольких, пустая строка для «все».
func promNamespaceMatcher(scope nsScope) string {
	switch len(scope.names) {
	case 0:
		return ""
	case 1:
		return fmt.Sprintf(`namespace="%s"`, promLabelValue(scope.names[0]))
	}
	quoted := make([]string, len(scope.names))
	for i, n := range scope.names {
		quoted[i] = regexp.QuoteMeta(n)
	}
	return fmt.Sprintf(`namespace=~"%s"`, promLabelValue(strings.Join(quoted, "|")))
}
