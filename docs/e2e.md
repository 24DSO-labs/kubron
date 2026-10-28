# Smoke-e2e веб-версии

Проверяет сквозной путь: браузер → `kubron-server` → (опционально) кластер. Файл: `frontend/e2e/smoke.spec.ts`.

## Что проверяется
1. Первый запуск: создание администратора, вход в интерфейс.
2. Cookie сессии (`HttpOnly`, `SameSite=Lax`), экран входа после потери cookie.
3. Защита API: 401 без сессии, 403 без CSRF-токена, 403 при чужом Origin и для закрытых методов, `AuthStatus` без сессии не раскрывает пользователя.
4. (Нужен `E2E_KUBECONFIG`) добавление кластера текстом через диалог, `ConnectToCluster`, `GetNamespaces` содержит `kube-system`.
5. (Нужны `E2E_KUBECONFIG` и `E2E_TERMINAL_POD=namespace/pod/container`, в образе должен быть `/bin/sh`) терминал: `OpenTerminal` выдаёт
   серверный id, по бинарному WebSocket `/api/terminal/{id}` команда выполняется в поде; `StartTerminalSession` с id клиента закрыт.

6. (`two-users.spec.ts`, gate этапа 2) два пользователя одновременно, у каждого своя сессия: оба видят себя в `AuthStatus`, права по роли
   (`ListUsers`: admin 200, user 403), в журнале аудита входы записаны от имени каждого с адресом, активный кластер у каждого свой
   (нужен `E2E_KUBECONFIG`), чужой терминал недоступен (нужен `E2E_TERMINAL_POD`), события логов приходят только владельцу потока
   (нужен `E2E_LOG_POD=namespace/pod/container` — под, пишущий в stdout), выход одного не затрагивает другого.
   Файл выполняется после `smoke.spec.ts` (по алфавиту) и использует созданные им администратора и кластер `e2e-kind`.

## Локальный запуск
```bash
cd frontend && npm ci && npm run build:web && cd ..
go build -o kubron-server ./cmd/kubron-server
KUBRON_DATA_DIR="$(mktemp -d)" ./kubron-server --listen 127.0.0.1:8080 &
cd frontend && npx playwright install chromium
E2E_KUBECONFIG=/path/to/kubeconfig npm run test:e2e   # без переменной тест кластера пропускается
```
PowerShell: `$env:KUBRON_DATA_DIR = New-Item -ItemType Directory -Path (Join-Path $env:TEMP "kubron-e2e-$(Get-Random)")`.
Каталог данных должен быть пустым — первый тест создаёт администратора.

## CI
Job `e2e` в `.github/workflows/ci.yml`: собирает веб-фронтенд и сервер, поднимает kind, запускает тесты.
Пока job помечен `continue-on-error: true`; после первого стабильного зелёного прогона флаг нужно убрать.
