// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

type item struct{ ns, name string }

func nsOfItem(i item) string { return i.ns }

func TestParseScope(t *testing.T) {
	t.Run("empty and blank names mean all", func(t *testing.T) {
		for _, in := range [][]string{nil, {}, {""}, {"  "}, {"a", ""}} {
			s, err := parseScope(in)
			if err != nil || !s.all() {
				t.Errorf("%q: expected 'all', got %v, %v", in, s.names, err)
			}
		}
		// невалидное имя отклоняется, даже если рядом есть пустое («все»)
		if _, err := parseScope([]string{"", "bad name"}); err == nil {
			t.Error("an invalid name must produce an error regardless of an empty one")
		}
	})

	t.Run("duplicates are removed, order is sorted", func(t *testing.T) {
		s, err := parseScope([]string{" b ", "a", "b", "c"})
		if err != nil || !reflect.DeepEqual(s.names, []string{"a", "b", "c"}) {
			t.Fatalf("%v %v", s.names, err)
		}
	})

	t.Run("invalid names are rejected", func(t *testing.T) {
		for _, bad := range []string{"Upper", "a_b", "a b", "-a", "a-", "a/b", "a\"b", "a.b", strings.Repeat("x", 64)} {
			if _, err := parseScope([]string{"ok", bad}); err == nil {
				t.Errorf("%q must be rejected", bad)
			}
		}
	})

	t.Run("size limit", func(t *testing.T) {
		var in []string
		for i := 0; i < maxScopeNamespaces; i++ {
			in = append(in, fmt.Sprintf("ns-%d", i))
		}
		if _, err := parseScope(in); err != nil {
			t.Fatalf("exactly at the limit must pass: %v", err)
		}
		if _, err := parseScope(append(in, "one-more")); err == nil {
			t.Error("over the limit is an error")
		}
		// дубли не раздувают набор
		if _, err := parseScope(append(append([]string{}, in...), in...)); err != nil {
			t.Errorf("duplicates must not be counted: %v", err)
		}
	})

	t.Run("error text is length-limited", func(t *testing.T) {
		_, err := parseScope([]string{strings.Repeat("Я", 500)})
		if err == nil || len([]rune(err.Error())) > 200 {
			t.Errorf("%v", err)
		}
	})
}

func fakeList(data map[string][]item, calls *[]string, mu *sync.Mutex) func(context.Context, string) ([]item, error) {
	return func(_ context.Context, ns string) ([]item, error) {
		if mu != nil {
			mu.Lock()
			*calls = append(*calls, ns)
			mu.Unlock()
		}
		if ns == "" {
			var all []item
			for _, k := range []string{"a", "b", "c", "d", "e", "f", "g"} {
				all = append(all, data[k]...)
			}
			return all, nil
		}
		items, ok := data[ns]
		if !ok {
			return nil, fmt.Errorf("forbidden: %s", ns)
		}
		return items, nil
	}
}

func TestListScopedSingleAndAll(t *testing.T) {
	data := map[string][]item{"a": {{"a", "1"}}, "b": {{"b", "1"}}}
	var calls []string
	var mu sync.Mutex
	list := fakeList(data, &calls, &mu)

	items, fails, err := listScoped(context.Background(), nsScope{}, false, list, nsOfItem)
	if err != nil || fails != nil || len(items) != 2 || !reflect.DeepEqual(calls, []string{""}) {
		t.Fatalf("all: %v %v %v calls=%v", items, fails, err, calls)
	}

	calls = nil
	items, _, err = listScoped(context.Background(), nsScope{names: []string{"b"}}, false, list, nsOfItem)
	if err != nil || len(items) != 1 || items[0].ns != "b" || !reflect.DeepEqual(calls, []string{"b"}) {
		t.Fatalf("one: %v %v calls=%v", items, err, calls)
	}

	// Ошибка одиночного namespace возвращается как ошибка, а не как «частичный» результат
	if _, fails, err := listScoped(context.Background(), nsScope{names: []string{"zzz"}}, false, list, nsOfItem); err == nil || fails != nil {
		t.Errorf("err=%v fails=%v", err, fails)
	}
}

func TestListScopedMergesDeterministically(t *testing.T) {
	data := map[string][]item{
		"a": {{"a", "x"}, {"a", "y"}},
		"b": {{"b", "x"}},
		"c": {{"c", "x"}},
	}
	// задержки в обратном порядке: результат не должен зависеть от того, кто закончил первым
	list := func(_ context.Context, ns string) ([]item, error) {
		time.Sleep(map[string]time.Duration{"a": 30, "b": 15, "c": 1}[ns] * time.Millisecond)
		return data[ns], nil
	}
	for i := 0; i < 5; i++ {
		items, fails, err := listScoped(context.Background(), nsScope{names: []string{"a", "b", "c"}}, false, list, nsOfItem)
		want := []item{{"a", "x"}, {"a", "y"}, {"b", "x"}, {"c", "x"}}
		if err != nil || fails != nil || !reflect.DeepEqual(items, want) {
			t.Fatalf("%v %v %v", items, fails, err)
		}
	}
}

func TestListScopedPartialFailure(t *testing.T) {
	data := map[string][]item{"a": {{"a", "1"}}, "c": {{"c", "1"}}}
	list := fakeList(data, nil, nil) // у "b" нет данных → ошибка

	items, fails, err := listScoped(context.Background(), nsScope{names: []string{"a", "b", "c"}}, false, list, nsOfItem)
	if err != nil {
		t.Fatalf("a partial denial must not be an error: %v", err)
	}
	if len(items) != 2 || len(fails) != 1 || fails[0].Namespace != "b" || !strings.Contains(fails[0].Error, "forbidden") {
		t.Errorf("items=%v fails=%v", items, fails)
	}

	// все упали → ошибка плюс полный список отказов
	items, fails, err = listScoped(context.Background(), nsScope{names: []string{"x", "y"}}, false, list, nsOfItem)
	if err == nil || len(fails) != 2 || items != nil {
		t.Errorf("items=%v fails=%v err=%v", items, fails, err)
	}
}

func TestListScopedEmptyResultIsNotNil(t *testing.T) {
	list := func(context.Context, string) ([]item, error) { return nil, nil }
	items, _, err := listScoped(context.Background(), nsScope{names: []string{"a", "b"}}, false, list, nsOfItem)
	if err != nil || items == nil || len(items) != 0 {
		t.Errorf("must be an empty but non-nil slice (JSON [] instead of null): %#v %v", items, err)
	}
}

func TestListScopedBoundsConcurrency(t *testing.T) {
	var cur, peak atomic.Int32
	list := func(_ context.Context, ns string) ([]item, error) {
		n := cur.Add(1)
		for {
			p := peak.Load()
			if n <= p || peak.CompareAndSwap(p, n) {
				break
			}
		}
		time.Sleep(10 * time.Millisecond)
		cur.Add(-1)
		return []item{{ns, "1"}}, nil
	}
	var names []string
	for i := 0; i < 30; i++ {
		names = append(names, fmt.Sprintf("ns-%02d", i))
	}
	items, _, err := listScoped(context.Background(), nsScope{names: names}, false, list, nsOfItem)
	if err != nil || len(items) != 30 {
		t.Fatalf("%d %v", len(items), err)
	}
	if peak.Load() > scopeConcurrency {
		t.Errorf("%d concurrent requests, limit %d", peak.Load(), scopeConcurrency)
	}
	if peak.Load() < 2 {
		t.Errorf("requests must run in parallel, peak %d", peak.Load())
	}
}

func TestListScopedCancel(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	started := make(chan struct{}, 64)
	list := func(ctx context.Context, ns string) ([]item, error) {
		started <- struct{}{}
		<-ctx.Done()
		return nil, ctx.Err()
	}
	var names []string
	for i := 0; i < 20; i++ {
		names = append(names, fmt.Sprintf("ns-%02d", i))
	}
	done := make(chan error, 1)
	go func() {
		_, _, err := listScoped(ctx, nsScope{names: names}, false, list, nsOfItem)
		done <- err
	}()
	<-started
	cancel()
	select {
	case err := <-done:
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("the call did not finish after the context was canceled")
	}
}

func TestListScopedListAllStrategy(t *testing.T) {
	data := map[string][]item{}
	for _, k := range []string{"a", "b", "c", "d", "e", "f", "g"} {
		data[k] = []item{{k, "1"}}
	}
	six := nsScope{names: []string{"a", "b", "c", "d", "e", "f"}}

	t.Run("a large set and the list-all permission: a single filtered request", func(t *testing.T) {
		var calls []string
		var mu sync.Mutex
		items, _, err := listScoped(context.Background(), six, true, fakeList(data, &calls, &mu), nsOfItem)
		if err != nil || len(items) != 6 || !reflect.DeepEqual(calls, []string{""}) {
			t.Fatalf("items=%d err=%v calls=%v", len(items), err, calls)
		}
		for _, it := range items {
			if it.ns == "g" {
				t.Error("a namespace outside the set must not appear in the response")
			}
		}
	})

	t.Run("a set not above the threshold: one request per namespace", func(t *testing.T) {
		var calls []string
		var mu sync.Mutex
		five := nsScope{names: six.names[:scopeListAllThreshold]}
		if _, _, err := listScoped(context.Background(), five, true, fakeList(data, &calls, &mu), nsOfItem); err != nil {
			t.Fatal(err)
		}
		if len(calls) != scopeListAllThreshold || calls[0] == "" {
			t.Errorf("calls=%v", calls)
		}
	})

	t.Run("no permission for the cluster-wide request: per-namespace fallback", func(t *testing.T) {
		var calls []string
		var mu sync.Mutex
		inner := fakeList(data, &calls, &mu)
		list := func(ctx context.Context, ns string) ([]item, error) {
			if ns == "" {
				mu.Lock()
				calls = append(calls, "")
				mu.Unlock()
				return nil, errors.New("forbidden: cluster-wide")
			}
			return inner(ctx, ns)
		}
		items, fails, err := listScoped(context.Background(), six, true, list, nsOfItem)
		if err != nil || fails != nil || len(items) != 6 {
			t.Fatalf("items=%d fails=%v err=%v", len(items), fails, err)
		}
		if calls[0] != "" || len(calls) != 7 {
			t.Errorf("calls=%v", calls)
		}
	})
}

func TestPromNamespaceMatcher(t *testing.T) {
	cases := []struct {
		in   []string
		want string
	}{
		{nil, ``},
		{[]string{"default"}, `namespace="default"`},
		{[]string{"a", "b-c"}, `namespace=~"a|b-c"`},
	}
	for _, c := range cases {
		s, err := parseScope(c.in)
		if err != nil {
			t.Fatal(err)
		}
		if got := promNamespaceMatcher(s); got != c.want {
			t.Errorf("%v: got %q, want %q", c.in, got, c.want)
		}
	}
	// регулярные метасимволы не должны проходить без экранирования даже в обход parseScope
	if got := promNamespaceMatcher(nsScope{names: []string{"a.b", "c"}}); got != `namespace=~"a\\.b|c"` {
		t.Errorf("got %q", got)
	}
}
