-- Состояние соединения кластеров (задача 3.6): результат фоновой проверки, общий для всех реплик.
-- Запись одна на кластер; строки удаляются вместе с кластером.
CREATE TABLE cluster_health (
    cluster_id text        PRIMARY KEY REFERENCES clusters (id) ON DELETE CASCADE,
    state      text        NOT NULL CHECK (state IN ('ok', 'unreachable', 'unauthorized', 'misconfigured', 'unknown')),
    message    text        NOT NULL DEFAULT '',
    latency_ms bigint      NOT NULL DEFAULT 0,
    checked_at timestamptz NOT NULL DEFAULT now()
);
