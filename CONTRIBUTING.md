# Участие в разработке Kubron

Спасибо, что помогаете проекту. Кратко о правилах.

## Лицензия и DCO

Ядро Kubron распространяется по лицензии [Apache-2.0](LICENSE). Отправляя изменение, вы
подтверждаете, что имеете право его передать и согласны на лицензирование по Apache-2.0
в соответствии с [Developer Certificate of Origin](https://developercertificate.org/).

Подтверждение — строка `Signed-off-by` в каждом коммите. Git добавляет её сам:

```
git commit -s -m "feat: ..."
```

Забыли? Для последнего коммита — `git commit --amend -s --no-edit`; для нескольких —
`git rebase --signoff <база>`. CI проверяет подписи в pull request (`scripts/check-dco.sh`).

Отдельное соглашение (CLA) не требуется.

## Заголовки в исходниках

В каждом `.go`, `.ts` и `.vue` файле первой строкой идёт заголовок:

```
// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0
```

Добавить автоматически: `python3 scripts/license-headers.py --fix`. CI без заголовка падает.

## Зависимости

CI проверяет лицензии зависимостей: Go — `go-licenses`, фронтенд — `license-checker`.
Допустимы Apache-2.0, MIT, BSD-2/3-Clause, ISC (для Go ещё MPL-2.0). Зависимость с другой
лицензией (в том числе GPL/AGPL/SSPL) без обсуждения не добавляется.

## Проверки перед pull request

```
gofmt -l .
go vet ./... && go test ./...
cd frontend && npm test
```
