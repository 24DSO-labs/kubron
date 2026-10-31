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
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/VadimIl224/kubron/frontend"
	"github.com/VadimIl224/kubron/internal/app"
	"github.com/VadimIl224/kubron/internal/k8s"
	"github.com/VadimIl224/kubron/internal/pgstore"
	webplatform "github.com/VadimIl224/kubron/internal/platform/web"
	"github.com/VadimIl224/kubron/internal/secretbox"
	"github.com/VadimIl224/kubron/internal/server"
	"github.com/VadimIl224/kubron/internal/store"
)

func main() {
	if len(os.Args) > 1 && adminCommands[os.Args[1]] {
		ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
		err := runAdmin(ctx, os.Args[1], os.Args[2:], os.Stdout)
		stop()
		if err != nil {
			log.Fatalf("%s: %v", os.Args[1], err)
		}
		return
	}
	cfg, err := server.LoadConfig(os.Args[1:], os.Getenv, os.Stderr)
	if err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return
		}
		log.Fatalf("configuration: %v", err)
	}

	hostname, _ := os.Hostname()
	if h := os.Getenv("HOSTNAME"); h != "" {
		hostname = h // в поде совпадает с его именем
	}
	replicaID, err := server.ResolveReplicaID(cfg.ReplicaID, hostname)
	if err != nil {
		log.Fatalf("configuration: %v", err)
	}
	lifecycle := server.NewLifecycle(replicaID)

	log.Printf("kubron-server %s, replica %s", version, replicaID)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dist, err := fs.Sub(frontend.Dist, "dist")
	if err != nil {
		log.Fatalf("frontend: %v", err)
	}
	if err := server.CheckWebBuild(dist); err != nil {
		log.Fatalf("frontend: %v", err)
	}

	// Каталог данных задаётся до запуска ядра: хранилища открываются в OnStartup
	k8s.SetKubronDir(cfg.DataDir)
	// Веб-сервер не запускает команды из kubeconfig и не читает по ним файлы сервера (задача 3.7).
	k8s.SetConnectionPolicy(k8s.ConnectionPolicy{AllowExec: false, AllowLocalFiles: false})

	hub := server.NewHub()
	platform := &webplatform.Platform{}
	platform.SetEmitter(hub.Emit)
	platform.SetSessionEmitter(hub.EmitTo)
	core := app.NewApp(platform)
	if cfg.DatabaseURL != "" {
		pg, err := openPostgres(ctx, cfg)
		if err != nil {
			log.Fatalf("database: %v", err)
		}
		defer pg.Close()
		app.SetStore(core, store.FromPostgres(pg))
	}
	app.OnStartup(core)(ctx)
	defer app.OnShutdown(core)(context.Background())

	dispatcher, err := server.NewAppDispatcher(core)
	if err != nil {
		log.Fatalf("dispatcher: %v", err)
	}
	// У каждой веб-сессии свой вид ядра (пользователь, права, кластер, терминалы, логи, события):
	// одновременно могут работать несколько пользователей.
	sessions := server.NewSessions(cfg.SessionTTL, false)
	terminals := platform.Terminals()
	guard := &server.Guard{Sessions: sessions, TrustProxy: cfg.TrustProxy, Hub: hub, Terminals: terminals,
		TrustedProxyHops: cfg.TrustedProxyHops, TrustedProxies: cfg.TrustedProxies,
		AnonLimiter: server.NewRateLimiter(server.DefaultAnonRate, server.DefaultAnonBurst), Lifecycle: lifecycle}
	app.SetResourceLimits(core, k8s.Limits{Terminals: cfg.MaxTerminals, LogStreams: cfg.MaxLogStreams, PortForwards: cfg.MaxPortForwards})
	app.SetClusterIdleTTL(core, 2*time.Minute)
	pf := setupPortForward(cfg, core, sessions)
	guard.NewView = func(src string) any { return app.NewSessionView(core, src) }
	guard.Bind = func(v any, id string) { app.BindView(v.(*app.App), id) }
	sessions.Release = func(v any) { app.ReleaseView(v.(*app.App)) }
	sessions.OnEnd = guard.CloseSession // закрытая сессия закрывает и свои WebSocket
	go terminals.Run(ctx.Done(), 5*time.Second)
	app.StartHealthMonitor(ctx, core, cfg.ClusterCheckInterval)
	// Фоновая уборка веб-сессий: закрывает те, у которых ядро уже завершило пользовательскую сессию
	// (простой, отключение пользователя) или истёк срок, и освобождает их ресурсы и ссылки пула.
	go func() {
		t := time.NewTicker(10 * time.Second)
		defer t.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.C:
				sessions.Reap(func(v any) bool { return app.ViewActive(v.(*app.App)) })
				// Права могли измениться (перенос кластера, исключение из команды): закрываем то, что
				// осталось открытым у потерявших доступ. Вне блокировки реестра сессий.
				for _, v := range sessions.Views() {
					app.EnforceAccess(v.(*app.App))
				}
			}
		}
	}()
	defer terminals.CloseAll()

	var handler http.Handler = server.NewHandler(dispatcher, dist, guard, server.WithCSPReportOnly(cfg.CSPReportOnly))
	if pf != nil {
		go pf.Run(ctx)
		// Закрытая веб-сессия закрывает и свои потоки port-forward (ReleaseView → StopAll); дополнительно
		// владелец проверяется на каждом запросе.
		handler = server.WithPortForwardHosts(handler, pf.Handler(), pf.IsPortForwardHost)
	}
	// SIGTERM: draining (новые соединения не принимаются, /readyz смотрит на lifecycle.Draining()),
	// пауза для балансировщика, затем закрытие WebSocket кадром 1012 и остановка HTTP-сервера.
	plan := server.ShutdownPlan{Drain: guard.BeginDrain, DrainDelay: cfg.DrainDelay, CloseConns: guard.CloseConnections}
	if err := server.RunWith(ctx, cfg, handler, plan); err != nil {
		log.Fatalf("server: %v", err)
	}
}

// openPostgres подключается к БД, применяет миграции и готовит организацию по умолчанию.
func openPostgres(ctx context.Context, cfg server.Config) (*pgstore.Store, error) {
	keys, err := secretbox.ParseKeyring(cfg.SecretKey, cfg.PreviousSecretKeys...)
	if err != nil {
		return nil, err
	}
	pg, err := pgstore.Open(ctx, cfg.DatabaseURL, keys)
	if err != nil {
		return nil, err
	}
	res, err := pg.Migrate(ctx)
	if err != nil {
		pg.Close()
		return nil, err
	}
	if len(res.Applied) > 0 {
		log.Printf("database: applied migrations %v", res.Applied)
	}
	if len(res.Ahead) > 0 {
		log.Printf("database: the schema is newer than this build (versions %v); running in compatibility mode", res.Ahead)
	}
	if err := pg.Init(ctx); err != nil {
		pg.Close()
		return nil, err
	}
	return pg, nil
}
