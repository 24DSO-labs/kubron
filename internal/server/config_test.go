// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"io"
	"net"
	"net/http"
	"testing"
	"time"
)

func env(m map[string]string) func(string) string { return func(k string) string { return m[k] } }

func TestLoadConfigDefaultsEnvAndFlags(t *testing.T) {
	cfg, err := LoadConfig(nil, env(nil), io.Discard)
	if err != nil || cfg.Listen != "127.0.0.1:8080" || cfg.AllowRemote {
		t.Fatalf("по умолчанию: %+v %v", cfg, err)
	}
	cfg, err = LoadConfig(nil, env(map[string]string{"KUBRON_LISTEN": "127.0.0.1:9000"}), io.Discard)
	if err != nil || cfg.Listen != "127.0.0.1:9000" {
		t.Fatalf("env: %+v %v", cfg, err)
	}
	cfg, err = LoadConfig([]string{"--listen", "localhost:7000"}, env(map[string]string{"KUBRON_LISTEN": "127.0.0.1:9000"}), io.Discard)
	if err != nil || cfg.Listen != "localhost:7000" {
		t.Fatalf("флаг важнее env: %+v %v", cfg, err)
	}
	if _, err := LoadConfig([]string{"--unknown"}, env(nil), io.Discard); err == nil {
		t.Error("неизвестный флаг должен давать ошибку")
	}
}

func TestRemoteListenNeedsExplicitAllow(t *testing.T) {
	for _, addr := range []string{":8080", "0.0.0.0:8080", "192.168.1.5:8080", "[::]:8080"} {
		if _, err := LoadConfig([]string{"--listen", addr}, env(nil), io.Discard); err == nil {
			t.Errorf("%s без --allow-remote должен отклоняться", addr)
		}
		if _, err := LoadConfig([]string{"--listen", addr, "--allow-remote"}, env(nil), io.Discard); err != nil {
			t.Errorf("%s с --allow-remote: %v", addr, err)
		}
	}
	for _, addr := range []string{"127.0.0.1:1", "localhost:1", "[::1]:1"} {
		if _, err := LoadConfig([]string{"--listen", addr}, env(nil), io.Discard); err != nil {
			t.Errorf("loopback %s: %v", addr, err)
		}
	}
	if _, err := LoadConfig([]string{"--listen", "bad"}, env(nil), io.Discard); err == nil {
		t.Error("адрес без порта должен отклоняться")
	}
	cfg, err := LoadConfig(nil, env(map[string]string{"KUBRON_LISTEN": "0.0.0.0:80", "KUBRON_ALLOW_REMOTE": "1"}), io.Discard)
	if err != nil || !cfg.AllowRemote {
		t.Errorf("env allow-remote: %+v %v", cfg, err)
	}
}

func TestServeAndGracefulShutdown(t *testing.T) {
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- Serve(ctx, ln, testHandler(t)) }()

	resp, err := http.Get("http://" + ln.Addr().String() + "/healthz")
	if err != nil || resp.StatusCode != 200 {
		t.Fatalf("healthz: %v %v", resp, err)
	}
	resp.Body.Close()

	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("Serve после отмены: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("сервер не завершился после отмены контекста")
	}
}

func TestLoadConfigSessionAndProxy(t *testing.T) {
	cfg, err := LoadConfig(nil, env(nil), io.Discard)
	if err != nil || cfg.SessionTTL != 12*time.Hour || cfg.TrustProxy {
		t.Fatalf("умолчания: %+v %v", cfg, err)
	}
	cfg, err = LoadConfig([]string{"--session-ttl", "30m"}, env(map[string]string{"KUBRON_TRUST_PROXY": "1"}), io.Discard)
	if err != nil || cfg.SessionTTL != 30*time.Minute || !cfg.TrustProxy {
		t.Fatalf("флаги и env: %+v %v", cfg, err)
	}
	for _, bad := range []map[string]string{{"KUBRON_SESSION_TTL": "быстро"}, {"KUBRON_SESSION_TTL": "-1h"}} {
		if _, err := LoadConfig(nil, env(bad), io.Discard); err == nil {
			t.Errorf("%v: ждали ошибку", bad)
		}
	}
}

func TestLoadConfigDataDir(t *testing.T) {
	cfg, err := LoadConfig(nil, env(map[string]string{"KUBRON_DATA_DIR": "/srv/kubron"}), io.Discard)
	if err != nil || cfg.DataDir != "/srv/kubron" {
		t.Fatalf("env: %+v %v", cfg, err)
	}
	cfg, err = LoadConfig([]string{"--data-dir", "/tmp/x"}, env(map[string]string{"KUBRON_DATA_DIR": "/srv/kubron"}), io.Discard)
	if err != nil || cfg.DataDir != "/tmp/x" {
		t.Fatalf("флаг важнее env: %+v %v", cfg, err)
	}
	if cfg, _ = LoadConfig(nil, env(nil), io.Discard); cfg.DataDir != "" {
		t.Errorf("по умолчанию пусто: %q", cfg.DataDir)
	}
}

func TestLoadConfigResourceLimits(t *testing.T) {
	cfg, err := LoadConfig(nil, env(nil), io.Discard)
	if err != nil || cfg.MaxTerminals != 8 || cfg.MaxLogStreams != 20 || cfg.MaxPortForwards != 10 {
		t.Fatalf("значения по умолчанию: %+v %v", cfg, err)
	}
	cfg, err = LoadConfig([]string{"--max-log-streams", "5"},
		env(map[string]string{"KUBRON_MAX_TERMINALS": "3", "KUBRON_MAX_LOG_STREAMS": "9", "KUBRON_MAX_PORT_FORWARDS": "0"}), io.Discard)
	if err != nil || cfg.MaxTerminals != 3 || cfg.MaxLogStreams != 5 || cfg.MaxPortForwards != 0 {
		t.Fatalf("env и флаги (флаг важнее env, 0 — без ограничения): %+v %v", cfg, err)
	}
	if _, err := LoadConfig(nil, env(map[string]string{"KUBRON_MAX_TERMINALS": "много"}), io.Discard); err == nil {
		t.Error("нечисловое значение должно отклоняться")
	}
	if _, err := LoadConfig([]string{"--max-terminals", "-1"}, env(nil), io.Discard); err == nil {
		t.Error("отрицательный лимит должен отклоняться")
	}
}
