// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package app

// Platform — всё, что ядру нужно от оболочки, в которой оно запущено (ADR-0002).
// Реализации: Wails (internal/platform/wails) и, позже, HTTP-сервер. Ядро не импортирует
// Wails: это проверяет imports_test.go.
type Platform interface {
	// Emit отправляет событие фронтенду (cluster:changed, logs:* и т. д.).
	Emit(event string, data ...any)
	// PickFile показывает диалог выбора файла и возвращает путь ("" — отмена).
	PickFile(title string) (string, error)
	// OpenURL открывает ссылку в браузере пользователя.
	OpenURL(url string)
	// Capabilities сообщает, что доступно в этой оболочке (фронтенд скрывает недоступное).
	Capabilities() Capabilities
}

// Capabilities — возможности оболочки, в которой запущено ядро (ADR-0010).
// Десктоп: всё локальное доступно. Веб-сервер: часть возможностей отключена или иная.
type Capabilities struct {
	FileDialog      bool `json:"fileDialog"`      // нативный диалог выбора файла
	PortForward     bool `json:"portForward"`     // проброс портов
	NodeShell       bool `json:"nodeShell"`       // оболочка на узле
	LocalKubeconfig bool `json:"localKubeconfig"` // чтение kubeconfig с диска этой машины
	SSO             bool `json:"sso"`             // вход через SSO/OIDC (платный модуль, этап 6)
}

// nopPlatform — заглушка для тестов и режимов без оболочки.
type nopPlatform struct{}

func (nopPlatform) Emit(string, ...any)             {}
func (nopPlatform) PickFile(string) (string, error) { return "", nil }
func (nopPlatform) OpenURL(string)                  {}
func (nopPlatform) Capabilities() Capabilities      { return Capabilities{} }
