# Спайки этапа 0

Ветка `spike/shared-pool`. Статус проверки указан честно: что реально запускалось, а что нет.

## Спайк 1. Общий пул и сессионные виды `Manager` (ADR-0005)

Изменения: `internal/k8s/pool.go` (новый), `client.go` (`Manager` = вид на общий `connPool`; `ForSession`, `Detach`), одна строка в `apply_create_test.go`, тесты `pool_test.go`.

Идея: подключения (client-go, informers) лежат в общем пуле с refcount; у каждого вида свой `activeCtx`, терминалы и логи. Остановка informers кластера — когда его не держит ни один вид (раньше: «закрыть все остальные кластеры при активации»). Остальные методы `Manager` (136 штук через `GetActiveConnection`) не менялись.

Проверено: логика пула и видов — 7 тестов с `-race` (независимый активный кластер, общий кластер закрывается после ухода последнего вида, `Detach`, `RemoveConnection`, неудачная активация, создание подключения ровно один раз при 50 параллельных вызовах, 50 видов × 200 переключений). Прогон шёл во временном модуле с заглушками `ClusterConnection`/`TerminalSessionManager`, потому что в моём окружении нет Go 1.27.3 и модулей client-go.
НЕ проверено: компиляция реального `client.go` и существующих тестов пакета `internal/k8s`. Нужно: `go test -race ./internal/k8s/...`.

Известные ограничения прототипа:
- создание подключения идёт под блокировкой пула; для нативного oidc-password (сетевой вызов) нужен singleflight по ключу;
- нет TTL: informers останавливаются сразу при refs=0, а не через паузу (убирает «дребезг» при переключениях);
- `logMaxStreams = 50` остаётся глобальным (нужен лимит на пользователя);
- `rawConfig`/`LoadContexts` остаются per-view и читают локальный kubeconfig (в вебе нужно убрать, этап 3).

## Спайк 2. Нативный oidc-password (ADR-0006, раздел 7)

Файлы: `internal/oidcpassword/` (источник токена, разбор exec-блока `oidc-login`, тесты), `cmd/oidc-spike/` (проверка против настоящего Keycloak и кластера).

Проверено: пакет `oidcpassword` собран и протестирован отдельно (`go vet`, `go test -race`) на заглушке IdP: password-grant, кэш, обновление за 30 с до истечения, возврат к паролю при отозванном refresh_token, один запрос в IdP при 50 параллельных вызовах, ошибка без пароля и логина, подстановка Bearer, разбор `--grant-type=password` и отказ для authcode.
НЕ проверено: настоящий Keycloak (isso.mts.ru недоступен из облака) и настоящий кластер; компиляция `cmd/oidc-spike`.

Запуск у себя:

```
export KUBRON_SPIKE_PASSWORD='…'
go run ./cmd/oidc-spike --kubeconfig ~/.kube/config --user <логин>
go test -race ./internal/oidcpassword/...
```

Что смотреть в выводе: (1) токен получен, срок действия и grant=password; (2) kube-apiserver принял токен. Если Keycloak отвечает ошибкой — пришли текст (пароль в нём не появляется): типичные причины — у клиента `kubernetes` выключен Direct Access Grants, нужен `client-secret`, другой issuer в discovery.
Что ещё выяснить спайком: срок жизни id_token и refresh_token в realm mts, обновление в течение суток, поведение при MFA.
