# Этап 1: веб-оболочка (итоги)

| Задача | Состояние | Где |
|---|---|---|
| 1.1 `cmd/kubron-server`, конфиг | сделано | `cmd/kubron-server`, `internal/server/config.go` |
| 1.2 RPC-диспетчер + allowlist + контрактный тест | сделано | `internal/server/rpc.go`, `methods.go`, `contract_test.go` |
| 1.3 HTTP: ошибки, Origin, лимиты, SPA, healthz | сделано | `internal/server/http.go`, `server.go` |
| 1.4 Сессии: cookie, CSRF, Fetch Metadata | сделано (сессии в памяти, одна активная) | `internal/server/session.go` |
| 1.5 Event hub + `/api/ws` | сделано (без адресации) | `internal/server/hub.go`, `ws.go` |
| 1.6 Терминал по binary WebSocket | сделано: `OpenTerminal` (id выдаёт сервер) + `/api/terminal/{id}`, ввод/вывод бинарные кадры, resize управляющим кадром; буфер вывода до подключения, сборка забытых сессий | `internal/termbuf`, `internal/server/terminal_ws.go`, `frontend/src/platform/terminal*.ts` |
| 1.7 Веб-слой фронтенда | сделано | `frontend/src/platform/*.web.ts`, `.env.web` |
| 1.8 Smoke-e2e Playwright + kind | сделано; CI-job пока `continue-on-error` | `frontend/e2e`, `docs/e2e.md` |

## Отклонения от ADR
- ADR-0004: сессии в памяти процесса, не в БД (для HA — этап 3); активный кластер в сессии появится с `Session` в ядре (этап 2).
- ADR-0009: хаб без адресации (события получают все подписанные). ID **терминалов** в вебе теперь выдаёт сервер (128 бит), но ID **потоков логов** (`StartLogStream`) по-прежнему выбирает клиент, а владелец ресурса не проверяется (сессия пока одна). **Блокер многопользовательского режима** — первая задача этапа 2: адресация событий по сессии, серверные ID и владелец для логов, привязка терминала к сессии-владельцу.

## Известные пробелы веб-режима
- Проброс порта (`GetFreePort`, адрес `localhost`) и `OpenArgoUI` закрыты/не работают (нужен HTTP-прокси, этап 3).
- `SetupAdmin` публичен, пока нет пользователей: безопасно только на loopback; для сети нужен одноразовый код настройки в логе сервера.
- Helm: `AddHelmRepo`/`SearchHelmCharts` ходят с сервера по URL пользователя (SSRF) и используют общий каталог Helm.
- TLS сервер не терминирует: для сети нужен прокси (`--trust-proxy`), иначе cookie уходит открытым текстом.
- ~~Нет ограничения попыток входа по IP~~ — устарело: с этапа 3 (3.5) есть блокировка по адресу клиента (30 попыток за 15 минут, экспоненциальная до 5 минут), `SetupAdmin` и `ChangePassword` покрыты, анонимные RPC ограничены token-bucket по адресу (`docs/http-security.md`). Состояние в памяти процесса: при нескольких репликах оно не общее (HA — этап 4).

## Проверено при ревизии (2026-10-27)
- В `internal` нет импортов Wails вне `internal/platform/wails`; во фронтенде нет обращений к Wails вне `src/platform`.
- Все вызовы `callApp` из фронтенда классифицированы в `allowedMethods`/`deniedMethods`; закрытые методы вызываются только в режимах интерфейса, скрытых в вебе (кроме порт-форварда и OpenArgoUI).
- Код multi-namespace (`scope*.go`) работает через `Manager.GetActiveConnection()` — совместим с моделью «Manager = вид сессии» (ADR-0005).
- Новые Helm-методы проходят `authz` и классифицированы.
