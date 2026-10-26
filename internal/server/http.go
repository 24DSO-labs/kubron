// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"encoding/json"
	"errors"
	"io"
	"log"
	"mime"
	"net/http"
	"net/url"
	"strings"
	"time"

	"analog/internal/auth"
)

// maxRPCBody — предел тела запроса (манифесты YAML крупные, но не мегабайты).
const maxRPCBody = 4 << 20

type rpcError struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type rpcResponse struct {
	Result any       `json:"result"`
	Error  *rpcError `json:"error,omitempty"`
}

// MapError переводит ошибку метода в HTTP-статус и машинный код (единый формат ошибок, задача 1.3).
// Текст сообщения остаётся прежним: фронтенд показывает его так же, как в десктопе, а по тексту
// «требуется вход в систему» распознаёт потерю сессии.
func MapError(err error) (status int, code string) {
	var bad *BadArgsError
	switch {
	case errors.Is(err, auth.ErrNotAuthenticated):
		return http.StatusUnauthorized, "unauthenticated"
	case errors.Is(err, auth.ErrForbidden):
		return http.StatusForbidden, "forbidden"
	case errors.Is(err, ErrUnknownMethod):
		return http.StatusNotFound, "unknown_method"
	case errors.Is(err, ErrMethodDenied):
		return http.StatusForbidden, "method_denied"
	case errors.As(err, &bad):
		return http.StatusBadRequest, "bad_request"
	default:
		return http.StatusInternalServerError, "error"
	}
}

// publicMethods — методы, доступные без веб-сессии: первичная настройка, вход, стартовое
// состояние и возможности оболочки для экрана входа. Остальные требуют cookie и CSRF-токен.
var publicMethods = map[string]bool{
	"AuthStatus": true, "SetupAdmin": true, "Login": true, "GetCapabilities": true,
}

// RPCHandler обслуживает POST /api/rpc/{method}: тело — JSON-массив аргументов,
// ответ — {"result": …} или {"error": {"code", "message"}}.
//
// Защита: (1) источник запроса — тот же Origin (и Sec-Fetch-Site, если браузер его шлёт);
// (2) Content-Type: application/json (чужая страница не может отправить такой запрос без
// CORS-preflight, на который сервер не отвечает); (3) при заданном guard — веб-сессия в cookie
// и CSRF-токен в заголовке X-CSRF-Token для всех методов, кроме publicMethods.
// guard == nil отключает сессии (тесты).
func RPCHandler(d *Dispatcher, g *Guard) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "forbidden_origin", "запрос с чужого источника отклонён")
			return
		}
		if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "bad_request", "ожидается Content-Type: application/json")
			return
		}
		method := r.PathValue("method")

		var token string
		var sess Session
		var authed bool
		if g != nil {
			token = g.token(r)
			sess, authed = g.Sessions.Get(token)
			if !publicMethods[method] {
				if !authed {
					writeError(w, http.StatusUnauthorized, "unauthenticated", auth.ErrNotAuthenticated.Error())
					return
				}
				if !csrfEqual(r.Header.Get(csrfHeader), sess.CSRF) {
					writeError(w, http.StatusForbidden, "csrf", "неверный или отсутствующий CSRF-токен")
					return
				}
			}
		}

		body, err := io.ReadAll(http.MaxBytesReader(w, r.Body, maxRPCBody))
		if err != nil {
			writeError(w, http.StatusRequestEntityTooLarge, "bad_request", "тело запроса слишком большое или не прочитано")
			return
		}
		var args []json.RawMessage
		if len(strings.TrimSpace(string(body))) > 0 {
			if err := json.Unmarshal(body, &args); err != nil {
				writeError(w, http.StatusBadRequest, "bad_request", "тело запроса должно быть JSON-массивом аргументов")
				return
			}
		}
		result, err := d.Call(method, args)
		if err != nil {
			status, code := MapError(err)
			if status == http.StatusInternalServerError {
				log.Printf("rpc %s: %v", method, err)
			}
			if g != nil && code == "unauthenticated" && authed {
				g.Sessions.Delete(token) // ядро закрыло сессию (простой, выход): cookie больше не действует
				g.clearCookie(w, r)
			}
			writeError(w, status, code, err.Error())
			return
		}
		if g != nil {
			result = g.afterCall(w, r, method, result, token, authed)
		}
		writeJSON(w, http.StatusOK, rpcResponse{Result: result})
	})
}

// afterCall связывает результат метода с веб-сессией: вход открывает её, выход закрывает,
// а стартовое состояние без сессии не раскрывает пользователя (ядро пока одно на всех).
func (g *Guard) afterCall(w http.ResponseWriter, r *http.Request, method string, result any, token string, authed bool) any {
	switch method {
	case "Login", "SetupAdmin":
		if u, ok := result.(*auth.UserInfo); ok && u != nil {
			if authed {
				g.Sessions.Delete(token)
			}
			tok, sess := g.Sessions.Create(u.ID)
			g.setCookie(w, r, tok, time.Until(sess.Expires))
			w.Header().Set(csrfHeader, sess.CSRF)
		}
	case "Logout":
		if authed {
			g.Sessions.Delete(token)
		}
		g.clearCookie(w, r)
	case "AuthStatus":
		if st, ok := result.(*auth.Status); ok && st != nil && !authed && st.User != nil {
			cp := *st
			cp.User = nil
			return &cp
		}
	}
	return result
}

// SessionHandler обслуживает GET /api/session: даёт странице CSRF-токен текущей сессии
// (после перезагрузки страницы) и признак входа. Без сессии токена нет.
func SessionHandler(g *Guard) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "forbidden_origin", "запрос с чужого источника отклонён")
			return
		}
		out := map[string]any{"authenticated": false}
		if g != nil {
			if sess, ok := g.Sessions.Get(g.token(r)); ok {
				out = map[string]any{"authenticated": true, "csrfToken": sess.CSRF}
			}
		}
		writeJSON(w, http.StatusOK, out)
	})
}

// sameOrigin: запрос без Origin (не браузерный клиент, curl) пропускается; с Origin —
// только если его хост совпадает с Host запроса. Если браузер сообщает Sec-Fetch-Site,
// допустимы только same-origin и none.
func sameOrigin(r *http.Request) bool {
	switch r.Header.Get("Sec-Fetch-Site") {
	case "", "same-origin", "none":
	default:
		return false
	}
	origin := r.Header.Get("Origin")
	if origin == "" {
		return true
	}
	u, err := url.Parse(origin)
	return err == nil && u.Host == r.Host
}

func writeError(w http.ResponseWriter, status int, code, msg string) {
	writeJSON(w, status, rpcResponse{Error: &rpcError{Code: code, Message: msg}})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(v); err != nil {
		log.Printf("rpc: запись ответа: %v", err)
	}
}
