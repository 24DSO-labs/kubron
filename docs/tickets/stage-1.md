# Этап 1: веб-оболочка (итоги)

| Задача | Состояние | Где |
|---|---|---|
| 1.1 `cmd/kubron-server`, конфиг | сделано | `cmd/kubron-server`, `internal/server/config.go` |
| 1.2 RPC-диспетчер + allowlist + контрактный тест | сделано | `internal/server/rpc.go`, `methods.go`, `contract_test.go` |
| 1.3 HTTP: ошибки, Origin, лимиты, SPA, healthz | сделано | `internal/server/http.go`, `server.go` |
| 1.4 Сессии: cookie, CSRF, Fetch Metadata | сделано (сессии в памяти, одна активная) | `internal/server/session.go` |
| 1.5 Event hub + `/api/ws` | сделано (без адресации) | `internal/server/hub.go`, `ws.go` |
| 1.6 Терминал по binary WebSocket | **отложено**: терминал работает через события `terminal:data:<id>` (base64) | — |
| 1.7 Веб-слой фронтенда | сделано | `frontend/src/platform/*.web.ts`, `.env.web` |
| 1.8 Smoke-e2e Playwright + kind | сделано; CI-job пока `continue-on-error` | `frontend/e2e`, `docs/e2e.md` |

## Отклонения от ADR
- ADR-0003: терминал не на binary WS, а на JSON-событиях. Работает; бинарный канал — оптимизация (меньше накладных расходов, обратное давление). Решить: оставить и поправить ADR или сделать в этапе 2/3.
- ADR-0004: сессии в памяти процесса, не в БД (для HA — этап 3); активный кластер в сессии появится с `Session` в ядре (этап 2).
- ADR-0009: хаб без адресации; ID терминалов и логов генерирует клиент (`term_<uid>`). Допустимо, пока веб-сессия одна. **Блокер многопользовательского режима** — первая задача этапа 2.

## Известные пробелы веб-режима
- Проброс порта (`GetFreePort`, адрес `localhost`) и `OpenArgoUI` закрыты/не работают (нужен HTTP-прокси, этап 3).
- `SetupAdmin` публичен, пока нет пользователей: безопасно только на loopback; для сети нужен одноразовый код настройки в логе сервера.
- Helm: `AddHelmRepo`/`SearchHelmCharts` ходят с сервера по URL пользователя (SSRF) и используют общий каталог Helm.
- TLS сервер не терминирует: для сети нужен прокси (`--trust-proxy`), иначе cookie уходит открытым текстом.
- Нет ограничения попыток входа по IP (блокировка только по логину).

## Проверено при ревизии (2026-10-27)
- В `internal` нет импортов Wails вне `internal/platform/wails`; во фронтенде нет обращений к Wails вне `src/platform`.
- Все вызовы `callApp` из фронтенда классифицированы в `allowedMethods`/`deniedMethods`; закрытые методы вызываются только в режимах интерфейса, скрытых в вебе (кроме порт-форварда и OpenArgoUI).
- Код multi-namespace (`scope*.go`) работает через `Manager.GetActiveConnection()` — совместим с моделью «Manager = вид сессии» (ADR-0005).
- Новые Helm-методы проходят `authz` и классифицированы.
