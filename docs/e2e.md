# Smoke-e2e веб-версии

Проверяет сквозной путь: браузер → `kubron-server` → (опционально) кластер. Файл: `frontend/e2e/smoke.spec.ts`.

## Что проверяется
1. Первый запуск: создание администратора, вход в интерфейс.
2. Cookie сессии (`HttpOnly`, `SameSite=Lax`), экран входа после потери cookie.
3. Защита API: 401 без сессии, 403 без CSRF-токена, 403 при чужом Origin и для закрытых методов, `AuthStatus` без сессии не раскрывает пользователя.
4. (Нужен `E2E_KUBECONFIG`) добавление кластера текстом через диалог, `ConnectToCluster`, `GetNamespaces` содержит `kube-system`.

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
