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
}

// nopPlatform — заглушка для тестов и режимов без оболочки.
type nopPlatform struct{}

func (nopPlatform) Emit(string, ...any)             {}
func (nopPlatform) PickFile(string) (string, error) { return "", nil }
func (nopPlatform) OpenURL(string)                  {}
