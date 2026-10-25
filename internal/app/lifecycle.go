package app

import "context"

// OnStartup и OnShutdown возвращают обработчики жизненного цикла для оболочки (Wails).
// Это функции пакета, а не методы App: экспортируемые методы App Wails привязывает к
// фронтенду, а они там не нужны (и подпадают под политику authz).
func OnStartup(a *App) func(context.Context) { return a.startup }

func OnShutdown(a *App) func(context.Context) { return a.shutdown }
