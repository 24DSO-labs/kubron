// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"time"
)

// Config — параметры сервера. Источники: флаги и переменные окружения (флаг важнее).
type Config struct {
	// Listen — адрес прослушивания, по умолчанию 127.0.0.1:8080 (KUBRON_LISTEN).
	Listen string
	// AllowRemote разрешает слушать не на loopback (KUBRON_ALLOW_REMOTE=1).
	// Без TLS cookie уходит открытым текстом: для сети нужен обратный прокси с TLS.
	AllowRemote bool
	// DataDir — каталог данных (пользователи, кластеры, журнал); пусто — ~/.kubron
	// (--data-dir или KUBRON_DATA_DIR). Читается только сервером: десктоп переменную игнорирует.
	DataDir string
	// TrustProxy — доверять X-Forwarded-Proto от обратного прокси: тогда cookie помечается
	// Secure (KUBRON_TRUST_PROXY=1). Включайте, только если сервер доступен лишь через прокси.
	TrustProxy bool
	// SessionTTL — абсолютный срок жизни веб-сессии (KUBRON_SESSION_TTL, по умолчанию 12h);
	// простой отслеживает ядро (SetIdleTimeout).
	SessionTTL time.Duration
	// Лимиты ресурсов на одного пользователя (0 — без ограничения):
	// KUBRON_MAX_TERMINALS, KUBRON_MAX_LOG_STREAMS, KUBRON_MAX_PORT_FORWARDS.
	MaxTerminals    int
	MaxLogStreams   int
	MaxPortForwards int
}

// Лимиты ресурсов на пользователя по умолчанию (ADR-0005).
const (
	DefaultMaxTerminals    = 8
	DefaultMaxLogStreams   = 20
	DefaultMaxPortForwards = 10
)

// LoadConfig разбирает флаги args (без имени программы) и окружение через getenv.
func LoadConfig(args []string, getenv func(string) string, out io.Writer) (Config, error) {
	cfg := Config{Listen: "127.0.0.1:8080", SessionTTL: 12 * time.Hour,
		MaxTerminals: DefaultMaxTerminals, MaxLogStreams: DefaultMaxLogStreams, MaxPortForwards: DefaultMaxPortForwards}
	if v := getenv("KUBRON_LISTEN"); v != "" {
		cfg.Listen = v
	}
	cfg.AllowRemote = truthy(getenv("KUBRON_ALLOW_REMOTE"))
	cfg.DataDir = getenv("KUBRON_DATA_DIR")
	cfg.TrustProxy = truthy(getenv("KUBRON_TRUST_PROXY"))
	if v := getenv("KUBRON_SESSION_TTL"); v != "" {
		d, err := time.ParseDuration(v)
		if err != nil {
			return Config{}, fmt.Errorf("KUBRON_SESSION_TTL %q: %w", v, err)
		}
		cfg.SessionTTL = d
	}

	for _, l := range []struct {
		env string
		dst *int
	}{
		{"KUBRON_MAX_TERMINALS", &cfg.MaxTerminals},
		{"KUBRON_MAX_LOG_STREAMS", &cfg.MaxLogStreams},
		{"KUBRON_MAX_PORT_FORWARDS", &cfg.MaxPortForwards},
	} {
		if v := getenv(l.env); v != "" {
			n, err := strconv.Atoi(strings.TrimSpace(v))
			if err != nil {
				return Config{}, fmt.Errorf("%s %q: %w", l.env, v, err)
			}
			*l.dst = n
		}
	}

	fs := flag.NewFlagSet("kubron-server", flag.ContinueOnError)
	fs.SetOutput(out)
	fs.StringVar(&cfg.Listen, "listen", cfg.Listen, "listen address (env KUBRON_LISTEN)")
	fs.BoolVar(&cfg.AllowRemote, "allow-remote", cfg.AllowRemote,
		"allow a non-loopback address; unsafe without a TLS proxy (env KUBRON_ALLOW_REMOTE)")
	fs.StringVar(&cfg.DataDir, "data-dir", cfg.DataDir, "data directory, defaults to ~/.kubron (env KUBRON_DATA_DIR)")
	fs.BoolVar(&cfg.TrustProxy, "trust-proxy", cfg.TrustProxy,
		"trust X-Forwarded-Proto from a reverse proxy, set Secure on cookies (env KUBRON_TRUST_PROXY)")
	fs.DurationVar(&cfg.SessionTTL, "session-ttl", cfg.SessionTTL, "web session lifetime (env KUBRON_SESSION_TTL)")
	fs.IntVar(&cfg.MaxTerminals, "max-terminals", cfg.MaxTerminals, "terminals per user, 0 for unlimited (env KUBRON_MAX_TERMINALS)")
	fs.IntVar(&cfg.MaxLogStreams, "max-log-streams", cfg.MaxLogStreams, "log streams per user, 0 for unlimited (env KUBRON_MAX_LOG_STREAMS)")
	fs.IntVar(&cfg.MaxPortForwards, "max-port-forwards", cfg.MaxPortForwards, "port-forwards per user, 0 for unlimited (env KUBRON_MAX_PORT_FORWARDS)")
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
	if c.SessionTTL <= 0 {
		return errors.New("session lifetime must be positive")
	}
	if c.MaxTerminals < 0 || c.MaxLogStreams < 0 || c.MaxPortForwards < 0 {
		return errors.New("resource limits cannot be negative (0 means unlimited)")
	}
	host, _, err := net.SplitHostPort(c.Listen)
	if err != nil {
		return fmt.Errorf("listen %q: %w", c.Listen, err)
	}
	if !c.AllowRemote && !isLoopbackHost(host) {
		return errors.New("address " + c.Listen + " is not loopback: the server does not terminate TLS, so the session cookie would be sent in plain text; " +
			"to allow network access, put a TLS proxy in front and pass --allow-remote")
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
