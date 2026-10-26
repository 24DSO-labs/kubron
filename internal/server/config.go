// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strings"
)

// Config — параметры сервера. Источники: флаги и переменные окружения (флаг важнее).
type Config struct {
	// Listen — адрес прослушивания, по умолчанию 127.0.0.1:8080 (KUBRON_LISTEN).
	Listen string
	// AllowRemote разрешает слушать не на loopback (KUBRON_ALLOW_REMOTE=1).
	// НЕБЕЗОПАСНО до появления сессий с cookie и CSRF (задача 1.4): сейчас сервер работает
	// с одной общей сессией, и любой, кто достучится до порта, действует от её имени.
	AllowRemote bool
}

// LoadConfig разбирает флаги args (без имени программы) и окружение через getenv.
func LoadConfig(args []string, getenv func(string) string, out io.Writer) (Config, error) {
	cfg := Config{Listen: "127.0.0.1:8080"}
	if v := getenv("KUBRON_LISTEN"); v != "" {
		cfg.Listen = v
	}
	cfg.AllowRemote = truthy(getenv("KUBRON_ALLOW_REMOTE"))

	fs := flag.NewFlagSet("kubron-server", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "адрес прослушивания (env KUBRON_LISTEN)")
	fs.BoolVar(&cfg.AllowRemote, "allow-remote", cfg.AllowRemote,
		"разрешить не-loopback адрес; НЕБЕЗОПАСНО без сессий (env KUBRON_ALLOW_REMOTE)")
	if err := fs.Parse(args); err != nil {
		return Config{}, err
	}
	return cfg, cfg.Validate()
}

func truthy(s string) bool {
	switch strings.ToLower(strings.TrimSpace(s)) {
	case "1", "true", "yes", "on":
		return true
	}
	return false
}

// Validate проверяет конфигурацию. Не-loopback адрес без явного разрешения — ошибка.
func (c Config) Validate() error {
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("listen %q: %w", c.Listen, err)
	}
	if !c.AllowRemote && !isLoopbackHost(host) {
		return errors.New("адрес " + c.Listen + " не loopback: сервер пока работает с одной общей сессией " +
			"и без аутентификации по cookie; для доступа по сети нужен --allow-remote (небезопасно)")
	}
	return nil
}

// isLoopbackHost: пустой host (":8080") означает все интерфейсы, это не loopback.
func isLoopbackHost(host string) bool {
	if host == "localhost" {
		return true
	}
	ip := net.ParseIP(host)
	return ip != nil && ip.IsLoopback()
}
