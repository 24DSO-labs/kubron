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

## Задача 1.5 (события по WebSocket) — сделано
- `internal/server/hub.go`: `Hub` (приёмник `webplatform.SetEmitter`), подписки по точным именам событий
  (`cluster:changed`, `terminal:data:<id>`, `logs:data:<id>`…), кадр сервера `{"event","data":[...]}`,
  очередь 256 кадров; медленный клиент отключается (клиент переподключается и перечитывает состояние).
- `internal/server/ws.go` (`GET /api/ws`, gorilla/websocket): нужна веб-сессия (cookie) и Origin того же хоста;
  сервер шлёт ping каждые 25 с, проверяет, что сессия жива; закрытие сессии закрывает её сокеты (`Sessions.OnEnd`).
- Протокол клиента: `{"op":"sub","events":[...]}` → `sub_ok`; `unsub` → `unsub_ok`; `ping` → `pong`; после подключения `ready`.
  Запускать поток (терминал, логи) нужно после `sub_ok`, иначе потеряются первые данные.
- Сейчас событие получают все подписчики (ядро одно на всех); адресация по сессии — этап 2.
- `go.mod`: gorilla/websocket стал прямой зависимостью (был indirect через Wails); после `go mod tidy` проверь diff.

## Задача 1.7 (веб-слой фронтенда) — сделано
- Транспорт выбирается при сборке (`VITE_TARGET=web`, файл `frontend/.env.web`, режим `--mode web`):
  `platform/api.ts` → `rpc.web.ts` (POST /api/rpc, cookie, `X-CSRF-Token`, повтор при 403 csrf, 401 → сброс сессии),
  `platform/events.ts` → `events.web.ts` (общий WebSocket, подсчёт подписок, переподключение с паузой,
  псевдособытие `platform:reconnected`, `flushSubscriptions()` перед каждым RPC, чтобы поток терминала/логов не обогнал подписку),
  `platform/files.ts` → `window.open` и `pickTextFile()` (загрузка kubeconfig браузером).
- Диалоги добавления/правки кластера без нативного выбора файла (`capabilities.fileDialog=false`): режим «Вставить текст» + «Загрузить из файла…».
- Команды: `npm run build:web` (vue-tsc + vite; нужен `wails generate module` для типов), затем `go build ./cmd/kubron-server`;
  разработка: `kubron-server` на :8080 + `npm run dev:web` (прокси /api и WebSocket).
- Сборка `vite build --mode web` проверена без wailsjs: Wails-runtime подменяется пустышкой, в бандле нет `EventsOn`.
- Известные пробелы веб-режима (не часть 1.7): проброс порта (`GetFreePort`, открытие `localhost:<порт>` — адрес сервера, а не
  браузера; нужен HTTP-прокси, этап 3), `OpenArgoUI`, `PickKubeconfigFile`/`AddClusterFromFile` закрыты `deniedMethods`.

## Задача 1.8 (smoke-e2e) — сделано, требует первого прогона в CI
- `frontend/e2e/smoke.spec.ts` (Playwright), `frontend/playwright.config.ts`, `npm run test:e2e`, описание в `docs/e2e.md`.
- Тесты: первый запуск и создание админа; cookie (HttpOnly, SameSite=Lax); защита API (401/403 csrf/403 origin/закрытые методы,
  `AuthStatus` без сессии); с `E2E_KUBECONFIG` — добавление кластера текстом и `GetNamespaces` (kube-system).
- Тест защиты API проверен против настоящего кода `internal/server` (временный сервер с заглушкой ядра): проходит.
  UI-шаги (первый запуск, вход, диалог кластера) и сценарий с kind проверит первый прогон у тебя/в CI — селекторы могут потребовать правки.
- Каталог данных сервера: `--data-dir` / `KUBRON_DATA_DIR` (умолчание `~/.kubron`). Читает только `kubron-server` (через `k8s.SetKubronDir`); десктоп переменную игнорирует, чтобы терминал с выставленной переменной не подменял данные пользователя.
- CI: job `e2e` (kind + Playwright), пока `continue-on-error: true`.
- `kubron-server` отказывается стартовать с десктопным `frontend/dist` (`server.CheckWebBuild`, метка
  `<meta name="kubron-target" content="web">` в index.html веб-сборки): `wails build` перезаписывает `dist`,
  поэтому перед `go build ./cmd/kubron-server` всегда `npm run build:web`.

## Заметки по Helm в веб-режиме (этап 3/5)
- `AddHelmRepo/UpdateHelmRepos/SearchHelmCharts/GetHelmChartPackage` ходят по сети с сервера по URL от пользователя
  (нужно ограничение адресов — защита от SSRF — и учёт прокси) и используют общий каталог Helm (`helmHomeDir`): репозитории
  общие для всех пользователей. `AddHelmRepo/RemoveHelmRepo/UpdateHelmRepos` требуют `cluster.manage`.

## Задача 1.6 (терминал по WebSocket) — сделано
- Веб: RPC `OpenTerminal(ns, pod, container, shell, cols, rows) → id` (id 128 бит выдаёт сервер, ядро: `App.OpenTerminal`,
  `k8s.Manager.StartTerminalWithOutput`, интерфейс `k8s.TerminalOutput`), затем `GET /api/terminal/{id}` (WebSocket):
  бинарные кадры — ввод/вывод, текстовый `{"type":"resize","cols","rows"}`, закрытие 1000 «exit» — процесс завершился.
- Вывод до подключения сокета копится (256 КиБ), поэтому подписок с подтверждением для терминала не нужно. Не подключились за 30 с
  или отключились дольше 30 с назад — сессия закрывается (`termbuf.Registry.Sweep`, запускается в `kubron-server`). Закрытие
  веб-сессии закрывает её терминальные сокеты (`Guard.CloseSession`).
- В вебе закрыты `StartTerminalSession`, `WriteTerminal`, `ResizeTerminal` (id от клиента, ввод только по сокету); `CloseTerminal` открыт.
- Фронтенд: `platform/terminal.ts` (`openTerminal`: десктоп — события Wails, веб — `terminal.web.ts`), `TerminalView.vue` переведён на него.
- Проверка: сокетные тесты сервера (в т.ч. из 1.5) прогнаны с `-race` на настоящем коде gorilla/websocket; e2e-тест терминала
  выполняется в CI с busybox-подом (`E2E_TERMINAL_POD`). Логи (`logs:data:<id>`) пока идут событиями с клиентскими id (этап 2).

## Дальше: этап 2
План тикетов — `docs/tickets/stage-2.md` (сессионный вид ядра `ForSession`, многосессионный `auth.Service`, адресация событий,
серверные id и владелец логов, снятие `Sessions.Exclusive`, e2e на двух пользователях). Нужны три решения владельца (в конце файла).

### Статус этапа 2
- **2.1 + 2.2 сделаны (код, ждут прогона у владельца):** `auth.Service` — вид над общим состоянием (`NewSession`/`Release`),
  у каждой веб-сессии свой вид ядра (`app.NewSessionView`/`ReleaseView` — функции пакета, не методы `App`),
  диспетчер вызывает метод на виде сессии (`Dispatcher.CallOn`). Вход выполняется на новом виде, при неудаче вид освобождается.
  Смена роли/отключение/сброс пароля/удаление завершают все сессии пользователя; смена пароля — остальные его сессии.
  Блокировка перебора учитывает адрес клиента (`--trust-proxy` → последний `X-Forwarded-For`).
- **2.3 сделано (код, не компилировался в моём окружении — нужен ваш `go test ./internal/...`):** у вида сессии свой `k8s.Manager`
  (`ForSession`: свой активный кластер, терминалы, логи) и свой port-forward-менеджер; пул подключений общий.
  Лимиты на пользователя `k8s.Limits` (сервер: 8 терминалов / 20 потоков логов / 10 port-forward, `app.SetResourceLimits`;
  десктоп без ограничений, логи 50). `ReleaseView` останавливает ресурсы сессии и снимает отметку активного кластера.
- **2.4 сделано:** события вида адресуются владельцу (`Hub.EmitTo`, `app.BindView` после создания сессии; до входа адресата нет).
  События общих кешей (`cluster:changed`, `argo:changed`, `helm:changed`) получают все виды, у которых кластер активен
  (`Manager.emitCluster`, пул хранит держателей). Подписка на чужой id событий ничего не даёт: событие адресовано владельцу.
  Тесты: `hub_test.go` (EmitTo), `pool_test.go` (получатели). Код `internal/k8s` не компилировался у меня — нужен ваш `go test`.
- **2.5 сделано:** владелец терминала — веб-сессия (`termbuf.Hooks.Owner`, `OwnedBy`; чужой терминал → 404 как несуществующий).
  Потоки логов и так принадлежат виду (свои `logs`, события адресованы владельцу), поэтому серверные id для логов не нужны;
  id от клиента ограничен алфавитом/длиной (`validStreamID`).
- **2.6 сделано:** права и журнал уже шли через вид сессии; добавлен адрес клиента (`audit.Event.Source`, `auth.Actor.Source`),
  событие `SessionEnd` несёт адрес завершённой сессии. Тест двух пользователей с разными ролями: `TestAuditAttributesActionsToSessionUserAndSource`.
- **2.8 сделано частично:** `connPool.getOrCreate` — singleflight вне блокировки пула (медленный/oidc-кластер не блокирует остальные,
  ошибка общая для ожидающих и не кешируется, паника → ошибка); задержка остановки informers `idleTTL` (сервер: 2 мин, `app.SetClusterIdleTTL`).
  Логика пула проверена у меня изолированно с `-race`. **Не сделан** бюджет одновременных LIST: нагрузку на apiserver пока ограничивает
  `QPS/Burst` клиента на подключение (общий для всех сессий кластера); отдельный бюджет — по результатам нагрузочного теста в 2.10.
- `Sessions.Exclusive` пока `true`. Следующее: 2.7 (снять ограничение), затем 2.9, 2.10.
