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

	"analog/frontend"
	"analog/internal/app"
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

	platform := &webplatform.Platform{}
	core := app.NewApp(platform)
	app.OnStartup(core)(ctx)
	defer app.OnShutdown(core)(context.Background())

	dispatcher, err := server.NewAppDispatcher(core)
	if err != nil {
		log.Fatalf("диспетчер: %v", err)
	}
	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		log.Fatalf("фронтенд: %v", err)
	}

	if err := server.Run(ctx, cfg, server.NewHandler(dispatcher, dist)); err != nil {
		log.Fatalf("сервер: %v", err)
	}
}
