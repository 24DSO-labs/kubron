# Kubron: состояние работ (для продолжения в новом чате)

Обновлено: 2026-10-26. Репозиторий `VadimIl224/kubron`, рабочая ветка `refactor` (разрешено пушить прямо в неё).

## Цель
Kubron (Wails + Go + Vue) должен стать ещё и веб-приложением (сервер на Go, фронтенд тот же) без второй кодовой базы:
для одиночных пользователей и команд DevOps одной организации. План — документ «Kubron: план разработки веб-версии
и доработки десктопа» (Claude Docs, проект kubron); ADR — `docs/adr/`, тикеты этапа 0 — `docs/tickets/stage-0.md`.

## Принятые решения (кратко)
- Архитектура «одно ядро, две оболочки»: `internal/app` (ядро), `main.go` (Wails), `cmd/kubron-server` (HTTP).
- Веб: универсальный RPC `POST /api/rpc/{method}` + WebSocket для событий и терминала (ADR-0003).
- Один кластер — одна команда; SSO только для входа в Kubron, кластеры подключаются по конфигу.
- Сервер на PostgreSQL (внешний), HA с 2+ репликами; SQLite только local-режим.
- Лицензия: ядро Apache-2.0; платный только SSO/OIDC — отдельный модуль в приватном репозитории (`pkg/ext`).
- Kubeconfig с `exec`/`oidc-login` в вебе запрещён; планируется нативный `oidc-password` (не проверен на реальном Keycloak;
  рабочий kubeconfig `nemo-dev` использует `authcode`, для него нужен другой путь — решить до задачи 3.7).

## Сделано
- Этап 0 (0.1–0.10): ядро в `internal/app`, `Platform`, слой `frontend/src/platform/*`, `GetCapabilities`,
  интерфейсы хранилищ (`auth.Backend`, `audit.Backend`, `k8s.TokenStore`, `internal/store`), лицензия/DCO/заголовки/CI,
  публичный модуль `pkg/ext` (+ пример `examples/ext-authenticator`).
- Спайки: общий пул подключений и сессионные виды `Manager` (в `refactor`); пакет `internal/oidcpassword`
  (тесты на заглушке IdP; против Keycloak не проверялось).
- Этап 1.1–1.3: `cmd/kubron-server`, `internal/server` (RPC-диспетчер с allowlist, единый формат ошибок,
  защита по Origin/Content-Type), `internal/platform/web`. Проверено вручную: `GetCapabilities`, `AuthStatus` через curl.

## Дальше (этап 1)
- 1.4 сессии: cookie `HttpOnly; Secure; SameSite=Lax`, CSRF-токен, Origin, автовыход по `ReportActivity`.
- 1.5 event hub + WebSocket `/api/ws`; 1.6 терминал по binary WebSocket.
- 1.7 фронтенд web: `platform/api.web.ts`, `events.web.ts`, `files.web.ts`, сборка `VITE_TARGET=web`.
- 1.8 smoke-e2e (Playwright + kind). Потом этап 2: мульти-пользовательское ядро (`Session`, `ForSession`).

## Правила работы
- Новый экспортируемый метод `app.App` нужно сразу классифицировать в `internal/server/methods.go`
  (`allowedMethods` или `deniedMethods`), иначе `TestEveryAppMethodIsClassified` падает.
- Каждый экспортируемый метод `App` начинается с `a.authz(...)` (проверяет `internal/auth/policy_test.go`).
- Коммиты подписываются (`git commit -s`), в `.go/.ts/.vue` нужен заголовок (`python3 scripts/license-headers.py --fix`).
- На Windows: `go test ./internal/... ./cmd/...` (папка `build/bin/files` в игнорируемом каталоге ломает `./...`);
  `-race` без cgo недоступен.
- Окружение ассистента не может скачать Go-модули и Go 1.27.3: код ядра с `client-go` он не компилирует,
  поэтому после его правок нужен `go test` и `wails build` у владельца.

## Задача 1.4 (сессии) — сделано
- `internal/server/session.go`: реестр веб-сессий (в памяти только SHA-256 токена), cookie `HttpOnly`, `SameSite=Lax`,
  `Secure` (имя `__Host-kubron_session`) при TLS или `--trust-proxy` + `X-Forwarded-Proto: https`; срок `--session-ttl` (12h).
- CSRF: токен в заголовке `X-CSRF-Token` обязателен для всех методов, кроме `AuthStatus/SetupAdmin/Login/GetCapabilities`;
  выдаётся в заголовке ответа Login/SetupAdmin и через `GET /api/session` (после перезагрузки страницы).
- Origin + `Sec-Fetch-Site` (только same-origin/none), `Content-Type: application/json`.
- Ограничение до этапа 2: ядро хранит одну общую сессию, поэтому новый вход отзывает все прочие веб-сессии
  (`Sessions.Exclusive`), а `AuthStatus` без cookie не раскрывает пользователя. На этапе 2 `Exclusive` выключается.
- Для фронтенда (1.7): `api.web.ts` шлёт `credentials: same-origin` и `X-CSRF-Token`; токен берёт из ответа Login и `GET /api/session`.
