// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package server — HTTP-оболочка ядра Kubron (ADR-0003): универсальный RPC
// POST /api/rpc/{method} поверх методов app.App, раздача фронтенда и служебные ручки.
package server

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"sort"
)

// Ошибки диспетчера; HTTP-статус для них выбирает MapError.
var (
	ErrUnknownMethod = errors.New("неизвестный метод")
	ErrMethodDenied  = errors.New("метод недоступен в веб-режиме")
)

// BadArgsError — аргументы вызова не соответствуют сигнатуре метода.
type BadArgsError struct{ Msg string }

func (e *BadArgsError) Error() string { return e.Msg }

// Dispatcher вызывает экспортируемые методы объекта по имени с аргументами из JSON.
// Вызываются только методы из allowlist; остальные недоступны, даже если существуют.
type Dispatcher struct {
	methods map[string]reflect.Value
	denied  map[string]string
}

var errorType = reflect.TypeOf((*error)(nil)).Elem()

// NewDispatcher строит диспетчер для target. Каждый метод из allowed должен существовать,
// быть не variadic и возвращать не более (результат, error): иначе это ошибка конфигурации,
// и сервер не должен стартовать.
func NewDispatcher(target any, allowed []string, denied map[string]string) (*Dispatcher, error) {
	tv := reflect.ValueOf(target)
	d := &Dispatcher{methods: make(map[string]reflect.Value, len(allowed)), denied: denied}
	for _, name := range allowed {
		m := tv.MethodByName(name)
		if !m.IsValid() {
			return nil, fmt.Errorf("allowlist: у %T нет метода %s", target, name)
		}
		t := m.Type()
		if t.IsVariadic() {
			return nil, fmt.Errorf("allowlist: метод %s variadic — не поддерживается", name)
		}
		if t.NumOut() > 2 || (t.NumOut() == 2 && t.Out(1) != errorType) {
			return nil, fmt.Errorf("allowlist: метод %s должен возвращать (T), (error) или (T, error)", name)
		}
		d.methods[name] = m
	}
	return d, nil
}

// Methods — имена доступных методов (по алфавиту).
func (d *Dispatcher) Methods() []string {
	out := make([]string, 0, len(d.methods))
	for n := range d.methods {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Call вызывает метод. args — JSON-значения позиционных аргументов. Паника метода
// превращается в обычную ошибку: сервер не падает из-за одного запроса.
func (d *Dispatcher) Call(name string, args []json.RawMessage) (result any, err error) {
	m, ok := d.methods[name]
	if !ok {
		if _, denied := d.denied[name]; denied {
			return nil, fmt.Errorf("%w: %s", ErrMethodDenied, name)
		}
		return nil, fmt.Errorf("%w: %s", ErrUnknownMethod, name)
	}
	t := m.Type()
	if len(args) != t.NumIn() {
		return nil, &BadArgsError{fmt.Sprintf("%s: ожидается аргументов: %d, получено: %d", name, t.NumIn(), len(args))}
	}
	in := make([]reflect.Value, len(args))
	for i, raw := range args {
		v, err := decodeArg(raw, t.In(i))
		if err != nil {
			return nil, &BadArgsError{fmt.Sprintf("%s: аргумент %d: %v", name, i+1, err)}
		}
		in[i] = v
	}

	defer func() {
		if r := recover(); r != nil {
			result, err = nil, fmt.Errorf("внутренняя ошибка при выполнении %s", name)
		}
	}()
	out := m.Call(in)
	switch len(out) {
	case 0:
		return nil, nil
	case 1:
		if t.Out(0) == errorType {
			if e, _ := out[0].Interface().(error); e != nil {
				return nil, e
			}
			return nil, nil
		}
		return out[0].Interface(), nil
	default:
		if e, _ := out[1].Interface().(error); e != nil {
			return nil, e
		}
		return out[0].Interface(), nil
	}
}

var bytesType = reflect.TypeOf([]byte(nil))

// decodeArg разбирает один аргумент. []byte принимается и как base64-строка (стандарт encoding/json),
// и как массив чисел (так их отдаёт Wails-фронтенд для ввода терминала).
func decodeArg(raw json.RawMessage, typ reflect.Type) (reflect.Value, error) {
	ptr := reflect.New(typ)
	if len(bytes.TrimSpace(raw)) == 0 {
		return ptr.Elem(), nil
	}
	if typ == bytesType {
		var s string
		if json.Unmarshal(raw, &s) == nil {
			b, err := base64.StdEncoding.DecodeString(s)
			if err != nil {
				return reflect.Value{}, fmt.Errorf("ожидается base64-строка или массив байт: %v", err)
			}
			ptr.Elem().SetBytes(b)
			return ptr.Elem(), nil
		}
		var nums []byte
		var ints []int
		if err := json.Unmarshal(raw, &ints); err != nil {
			return reflect.Value{}, errors.New("ожидается base64-строка или массив байт")
		}
		for _, n := range ints {
			if n < 0 || n > 255 {
				return reflect.Value{}, errors.New("значение байта вне диапазона 0..255")
			}
			nums = append(nums, byte(n))
		}
		ptr.Elem().SetBytes(nums)
		return ptr.Elem(), nil
	}
	if err := json.Unmarshal(raw, ptr.Interface()); err != nil {
		return reflect.Value{}, err
	}
	return ptr.Elem(), nil
}
