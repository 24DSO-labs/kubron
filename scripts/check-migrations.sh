#!/usr/bin/env bash
# Проверка SQL-миграций на настоящем PostgreSQL (CI, задача 3.1a): вперёд → назад → вперёд → назад.
# Каждая миграция применяется в транзакции; после полного отката в схеме не остаётся таблиц.
# Использование: PGHOST=... PGUSER=... PGPASSWORD=... PGDATABASE=... scripts/check-migrations.sh
set -euo pipefail

dir="$(cd "$(dirname "$0")/.." && pwd)/internal/pgstore/migrations"
psql_() { psql -v ON_ERROR_STOP=1 -q -t -A "$@"; }
tables() { psql_ -c "select count(*) from information_schema.tables where table_schema = 'public'"; }

up() {
  for f in $(ls "$dir"/*.up.sql | sort); do
    echo "up   $(basename "$f")"
    { echo "BEGIN;"; cat "$f"; echo "COMMIT;"; } | psql_ -f -
  done
}
down() {
  for f in $(ls "$dir"/*.down.sql | sort -r); do
    echo "down $(basename "$f")"
    { echo "BEGIN;"; cat "$f"; echo "COMMIT;"; } | psql_ -f -
  done
}

if [ "$(tables)" != "0" ]; then
  echo "ожидалась пустая схема public, таблиц: $(tables)" >&2
  exit 1
fi
for round in 1 2; do
  echo "== раунд $round"
  up
  n=$(tables)
  [ "$n" -gt 0 ] || { echo "после up нет таблиц" >&2; exit 1; }
  down
  n=$(tables)
  [ "$n" = "0" ] || { echo "после down осталось таблиц: $n" >&2; exit 1; }
done
echo "миграции применяются и откатываются"
