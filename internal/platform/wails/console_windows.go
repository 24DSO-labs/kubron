// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

//go:build windows

package wailsplatform

import "syscall"

// HideChildConsole создаёт для процесса скрытую консоль.
//
// GUI-приложение на Windows запускается без консоли, и каждый консольный дочерний процесс
// (kubectl oidc-login, который client-go запускает для exec-аутентификации) получает своё
// видимое окно cmd. Если у процесса есть скрытая консоль, дети наследуют её и окон не создают.
// Окно консоли (если есть, например в `wails dev`) не трогаем.
func HideChildConsole() {
	kernel32 := syscall.NewLazyDLL("kernel32.dll")
	user32 := syscall.NewLazyDLL("user32.dll")
	getConsoleWindow := kernel32.NewProc("GetConsoleWindow")
	if hwnd, _, _ := getConsoleWindow.Call(); hwnd != 0 {
		return
	}
	if ok, _, _ := kernel32.NewProc("AllocConsole").Call(); ok == 0 {
		return
	}
	if hwnd, _, _ := getConsoleWindow.Call(); hwnd != 0 {
		const swHide = 0
		user32.NewProc("ShowWindow").Call(hwnd, swHide)
	}
}
