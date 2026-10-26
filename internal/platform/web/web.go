// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package webplatform — реализация ext.Platform для HTTP-оболочки (kubron-server).
package webplatform

import (
	"errors"
	"sync"

	"analog/pkg/ext"
)

// ErrNotSupported — операция есть только в десктопе.
var ErrNotSupported = errors.New("недоступно в веб-режиме")

// Platform: события уходят подписчикам через эмиттер (WebSocket-хаб, задача 1.5);
// пока эмиттер не задан, события отбрасываются.
type Platform struct {
	mu   sync.RWMutex
	emit func(event string, data ...any)
}

// SetEmitter подключает доставку событий браузеру.
func (p *Platform) SetEmitter(f func(event string, data ...any)) {
	p.mu.Lock()
	p.emit = f
	p.mu.Unlock()
}

func (p *Platform) Emit(event string, data ...any) {
	p.mu.RLock()
	f := p.emit
	p.mu.RUnlock()
	if f != nil {
		f(event, data...)
	}
}

// PickFile: нативного диалога на сервере нет; браузер загружает файл сам (<input type=file>).
func (p *Platform) PickFile(string) (string, error) { return "", ErrNotSupported }

// OpenURL: ссылки в вебе открывает сам браузер.
func (p *Platform) OpenURL(string) {}

// Capabilities веб-оболочки. FileDialog и LocalKubeconfig выключены: kubeconfig передаётся
// текстом, а не путём на файловой системе сервера.
func (p *Platform) Capabilities() ext.Capabilities {
	return ext.Capabilities{
		FileDialog:      false,
		PortForward:     true,
		NodeShell:       true,
		LocalKubeconfig: false,
		SSO:             false,
	}
}
