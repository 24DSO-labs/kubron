// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"reflect"
	"sort"
	"testing"

	"analog/internal/app"
)

// Каждый экспортируемый метод app.App обязан быть либо в allowlist, либо явно в списке запрещённых:
// так новый метод не попадает в веб незамеченным, а решение принимается осознанно.
func TestEveryAppMethodIsClassified(t *testing.T) {
	allowed := map[string]bool{}
	for _, n := range allowedMethods {
		if allowed[n] {
			t.Errorf("метод %s в allowlist дважды", n)
		}
		allowed[n] = true
	}
	for n := range deniedMethods {
		if allowed[n] {
			t.Errorf("метод %s одновременно в allowlist и в списке запрещённых", n)
		}
	}

	typ := reflect.TypeOf(&app.App{})
	var unclassified []string
	exists := map[string]bool{}
	for i := 0; i < typ.NumMethod(); i++ {
		name := typ.Method(i).Name
		exists[name] = true
		if _, denied := deniedMethods[name]; !allowed[name] && !denied {
			unclassified = append(unclassified, name)
		}
	}
	sort.Strings(unclassified)
	if len(unclassified) > 0 {
		t.Errorf("методы App без решения (добавьте в allowedMethods или deniedMethods в methods.go): %v", unclassified)
	}
	for n := range allowed {
		if !exists[n] {
			t.Errorf("в allowlist метод %s, которого нет у App (устаревшая запись)", n)
		}
	}
	for n := range deniedMethods {
		if !exists[n] {
			t.Errorf("в списке запрещённых метод %s, которого нет у App (устаревшая запись)", n)
		}
	}
}

// Сигнатуры всех разрешённых методов поддерживаются диспетчером (не variadic, ≤ 2 результатов).
func TestAppDispatcherBuilds(t *testing.T) {
	d, err := NewAppDispatcher(app.NewApp(nil))
	if err != nil {
		t.Fatal(err)
	}
	if got := len(d.Methods()); got != len(allowedMethods) {
		t.Errorf("в диспетчере %d методов, в allowlist %d", got, len(allowedMethods))
	}
}

// До входа публичные методы работают через RPC, остальные отказывают — без паники.
func TestAppRPCWithoutLogin(t *testing.T) {
	d, err := NewAppDispatcher(app.NewApp(nil))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := d.Call("GetClusterStatuses", nil); err != nil {
		t.Fatalf("GetClusterStatuses без входа возвращает пустой список, а не ошибку: %v", err)
	}
	if _, err := d.Call("GetNamespaces", nil); err == nil {
		t.Fatal("без входа GetNamespaces должен возвращать ошибку доступа")
	}
	caps, err := d.Call("GetCapabilities", nil)
	if err != nil || caps == nil {
		t.Fatalf("GetCapabilities публичный: %v %v", caps, err)
	}
}
