// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net"
	"net/http"
	"path"
	"strings"
	"time"
)

// NewHandler собирает маршруты: RPC, служебные ручки, статика фронтенда с SPA-fallback.
// g — защита веб-API (сессии, CSRF); nil отключает её. static — содержимое собранного фронтенда (корень с index.html); может быть nil.
func NewHandler(d *Dispatcher, static fs.FS, g *Guard) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("POST /api/rpc/{method}", RPCHandler(d, g))
	mux.Handle("GET /api/session", SessionHandler(g))
	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		_, _ = w.Write([]byte("ok\n"))
	})
	mux.HandleFunc("/api/", func(w http.ResponseWriter, _ *http.Request) {
		writeError(w, http.StatusNotFound, "not_found", "нет такой ручки API")
	})
	mux.Handle("/", spaHandler(static))
	return securityHeaders(mux)
}

func securityHeaders(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		h := w.Header()
		h.Set("X-Content-Type-Options", "nosniff")
		h.Set("X-Frame-Options", "DENY")
		h.Set("Referrer-Policy", "no-referrer")
		next.ServeHTTP(w, r)
	})
}

// spaHandler отдаёт файлы фронтенда; для любого пути без файла (клиентский роутинг) — index.html.
func spaHandler(static fs.FS) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet && r.Method != http.MethodHead {
			http.Error(w, "метод не поддерживается", http.StatusMethodNotAllowed)
			return
		}
		if static == nil {
			http.Error(w, "фронтенд не встроен в сборку", http.StatusServiceUnavailable)
			return
		}
		name := strings.TrimPrefix(path.Clean("/"+r.URL.Path), "/")
		if name == "" {
			name = "index.html"
		}
		if st, err := fs.Stat(static, name); err != nil || st.IsDir() {
			name = "index.html" // клиентский маршрут
		}
		f, err := static.Open(name)
		if err != nil {
			http.Error(w, "фронтенд не собран (нет index.html): выполните сборку frontend", http.StatusServiceUnavailable)
			return
		}
		defer f.Close()
		st, err := f.Stat()
		if err != nil {
			http.Error(w, "ошибка чтения", http.StatusInternalServerError)
			return
		}
		if strings.HasPrefix(name, "assets/") {
			w.Header().Set("Cache-Control", "public, max-age=31536000, immutable") // имена с хешем
		} else {
			w.Header().Set("Cache-Control", "no-cache")
		}
		rs, ok := f.(interface {
			Read([]byte) (int, error)
			Seek(int64, int) (int64, error)
		})
		if !ok {
			http.Error(w, "файл не поддерживает чтение с позиции", http.StatusInternalServerError)
			return
		}
		http.ServeContent(w, r, name, st.ModTime(), rs)
	})
}

// Run слушает cfg.Listen и обслуживает h до отмены ctx, затем корректно завершает соединения.
func Run(ctx context.Context, cfg Config, h http.Handler) error {
	if err := cfg.Validate(); err != nil {
		return err
	}
	ln, err := net.Listen("tcp", cfg.Listen)
	if err != nil {
		return err
	}
	return Serve(ctx, ln, h)
}

// Serve обслуживает listener до отмены ctx (graceful shutdown до 10 секунд).
func Serve(ctx context.Context, ln net.Listener, h http.Handler) error {
	srv := &http.Server{
		Handler:           h,
		ReadHeaderTimeout: 10 * time.Second,
		IdleTimeout:       120 * time.Second,
	}
	errc := make(chan error, 1)
	go func() { errc <- srv.Serve(ln) }()
	log.Printf("kubron-server слушает http://%s", ln.Addr())

	select {
	case err := <-errc:
		if errors.Is(err, http.ErrServerClosed) {
			return nil
		}
		return err
	case <-ctx.Done():
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		if err := srv.Shutdown(shutdownCtx); err != nil {
			return err
		}
		return nil
	}
}
