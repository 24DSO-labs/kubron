#!/usr/bin/env bash
# DCO: каждый коммит (кроме слияний) должен содержать строку "Signed-off-by: Имя <email>".
# Использование: scripts/check-dco.sh <база> <голова>   (git commit -s добавляет строку сам)
set -euo pipefail
base="${1:?base}"; head="${2:?head}"
bad=0
for sha in $(git rev-list --no-merges "$base..$head"); do
  if ! git show -s --format=%B "$sha" | grep -Eq '^Signed-off-by: .+ <.+@.+>$'; then
    echo "Нет Signed-off-by: $(git show -s --format='%h %s' "$sha")"
    bad=1
  fi
done
if [ "$bad" -ne 0 ]; then
  echo "Подпишите коммиты: git commit -s --amend (или git rebase --signoff $base). См. CONTRIBUTING.md"
  exit 1
fi
