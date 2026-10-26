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

// RPCHandler обслуживает POST /api/rpc/{method}: тело — JSON-массив аргументов,
// ответ — {"result": …} или {"error": {"code", "message"}}.
//
// Защита от чужих страниц (до появления cookie-сессий и CSRF-токена, задача 1.4): принимается
// только Content-Type: application/json (такой запрос из чужой страницы требует CORS-preflight,
// на который сервер не отвечает) и только с тем же Origin, что и Host.
func RPCHandler(d *Dispatcher) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !sameOrigin(r) {
			writeError(w, http.StatusForbidden, "forbidden_origin", "запрос с чужого источника отклонён")
			return
		}
		if mt, _, _ := mime.ParseMediaType(r.Header.Get("Content-Type")); mt != "application/json" {
			writeError(w, http.StatusUnsupportedMediaType, "bad_request", "ожидается Content-Type: application/json")
			return
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
		result, err := d.Call(r.PathValue("method"), args)
		if err != nil {
			status, code := MapError(err)
			if status == http.StatusInternalServerError {
				log.Printf("rpc %s: %v", r.PathValue("method"), err)
			}
			writeError(w, status, code, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, rpcResponse{Result: result})
	})
}

// sameOrigin: запрос без Origin (не браузерный клиент, curl) пропускается; с Origin —
// только если его хост совпадает с Host запроса.
func sameOrigin(r *http.Request) bool {
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
