// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// kubron-server — HTTP-оболочка Kubron: то же ядро и тот же интерфейс, что в десктопе,
// но в браузере. Сейчас — режим одного пользователя на loopback (этап 1 плана).
//
//	kubron-server --listen 127.0.0.1:8080
package main

import (
	"context"
	"errors"
	"flag"
	"io/fs"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"analog/frontend"
	"analog/internal/app"
	"analog/internal/k8s"
	webplatform "analog/internal/platform/web"
	"analog/internal/server"
)

func main() {
	cfg, err := server.LoadConfig(os.Args[1:], os.Getenv, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("конфигурация: %v", err)
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		log.Fatalf("фронтенд: %v", err)
	}
	if err := server.CheckWebBuild(dist); err != nil {
		log.Fatalf("фронтенд: %v", err)
	}

	// Каталог данных задаётся до запуска ядра: хранилища открываются в OnStartup
	k8s.SetKubronDir(cfg.DataDir)

	hub := server.NewHub()
	platform := &webplatform.Platform{}
	platform.SetEmitter(hub.Emit)
	platform.SetSessionEmitter(hub.EmitTo)
	core := app.NewApp(platform)
	app.OnStartup(core)(ctx)
	defer app.OnShutdown(core)(context.Background())

	dispatcher, err := server.NewAppDispatcher(core)
	if err != nil {
		log.Fatalf("диспетчер: %v", err)
	}
	// У каждой веб-сессии свой вид ядра (пользователь, права, кластер, терминалы, логи, события):
	// одновременно могут работать несколько пользователей.
	sessions := server.NewSessions(cfg.SessionTTL, false)
	terminals := platform.Terminals()
	guard := &server.Guard{Sessions: sessions, TrustProxy: cfg.TrustProxy, Hub: hub, Terminals: terminals}
	app.SetResourceLimits(core, k8s.DefaultLimits())
	app.SetClusterIdleTTL(core, 2*time.Minute)
	guard.NewView = func(src string) any { return app.NewSessionView(core, src) }
	guard.Bind = func(v any, id string) { app.BindView(v.(*app.App), id) }
	sessions.Release = func(v any) { app.ReleaseView(v.(*app.App)) }
	sessions.OnEnd = guard.CloseSession // закрытая сессия закрывает и свои WebSocket
	go terminals.Run(ctx.Done(), 5*time.Second)
	defer terminals.CloseAll()

	if err := server.Run(ctx, cfg, server.NewHandler(dispatcher, dist, guard)); err != nil {
		log.Fatalf("сервер: %v", err)
	}
}
