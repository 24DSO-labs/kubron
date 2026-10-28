-- Начальная схема сервисной модели «организация → команда → кластер» (этап 3, задача 3.1a).
-- Идентификаторы — text: у пользователей это hex (auth.newID), у кластеров — slug (k8s.clusterIDRe).

CREATE TABLE organizations (
    id         text        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT organizations_name_key UNIQUE (name)
);

CREATE TABLE teams (
    id         text        PRIMARY KEY DEFAULT gen_random_uuid()::text,
    org_id     text        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT teams_org_name_key UNIQUE (org_id, name)
);

CREATE TABLE users (
    id                   text        PRIMARY KEY,
    org_id               text        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    username             text        NOT NULL,
    password_hash        text        NOT NULL DEFAULT '',
    org_role             text        NOT NULL DEFAULT 'member' CHECK (org_role IN ('org-admin', 'member')),
    -- Переходное поле: глобальная роль до перехода authz на членство в командах (задача 3.2, там колонка удаляется).
    role                 text        NOT NULL DEFAULT 'user' CHECK (role IN ('admin', 'operator', 'user')),
    disabled             boolean     NOT NULL DEFAULT false,
    must_change_password boolean     NOT NULL DEFAULT false,
    -- Связь с внешним провайдером входа (SSO, этап 6); у локальных пользователей пусто.
    external_id          text,
    provider             text,
    created_at           timestamptz NOT NULL DEFAULT now(),
    updated_at           timestamptz NOT NULL DEFAULT now(),
    last_login_at        timestamptz,
    CONSTRAINT users_external_pair CHECK ((external_id IS NULL) = (provider IS NULL))
);
CREATE UNIQUE INDEX users_username_key ON users (org_id, lower(username));
CREATE UNIQUE INDEX users_external_key ON users (org_id, provider, external_id) WHERE external_id IS NOT NULL;

CREATE TABLE memberships (
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    team_id    text        NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    role       text        NOT NULL CHECK (role IN ('admin', 'operator', 'user')),
    created_at timestamptz NOT NULL DEFAULT now(),
    PRIMARY KEY (user_id, team_id)
);
CREATE INDEX memberships_team_idx ON memberships (team_id);

-- Один кластер — одна команда. Секреты (kubeconfig, пароль oidc-password) хранятся только зашифрованными
-- (AES-GCM, задача 3.1c); key_id указывает ключ, которым они зашифрованы.
CREATE TABLE clusters (
    id                text        PRIMARY KEY CHECK (id ~ '^[A-Za-z0-9_-]{1,80}$'),
    team_id           text        NOT NULL REFERENCES teams (id) ON DELETE RESTRICT,
    display_name      text        NOT NULL,
    icon              text        NOT NULL DEFAULT '',
    color             text        NOT NULL DEFAULT '',
    environment       text        NOT NULL DEFAULT '' CHECK (environment IN ('', 'dev', 'test', 'stage', 'prod')),
    default_namespace text        NOT NULL DEFAULT '',
    context_name      text        NOT NULL DEFAULT '',
    mode              text        NOT NULL DEFAULT 'kubeconfig' CHECK (mode IN ('kubeconfig', 'in-cluster', 'oidc-password')),
    kubeconfig_enc    bytea,
    secret_enc        bytea,
    key_id            text        NOT NULL DEFAULT '',
    created_by        text        REFERENCES users (id) ON DELETE SET NULL,
    added_at          timestamptz NOT NULL DEFAULT now(),
    updated_at        timestamptz NOT NULL DEFAULT now(),
    CONSTRAINT clusters_cipher_key CHECK ((kubeconfig_enc IS NULL AND secret_enc IS NULL) OR key_id <> ''),
    CONSTRAINT clusters_kubeconfig_mode CHECK (mode = 'in-cluster' OR kubeconfig_enc IS NOT NULL)
);
CREATE INDEX clusters_team_idx ON clusters (team_id);
CREATE UNIQUE INDEX clusters_team_name_key ON clusters (team_id, lower(display_name));

CREATE TABLE argo_tokens (
    cluster_id text        PRIMARY KEY REFERENCES clusters (id) ON DELETE CASCADE,
    token_enc  bytea       NOT NULL,
    key_id     text        NOT NULL CHECK (key_id <> ''),
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Соответствие группы внешнего провайдера команде и роли (редактор в UI — этап 6); пока пустая.
CREATE TABLE group_mappings (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    provider   text   NOT NULL,
    group_name text   NOT NULL,
    team_id    text   NOT NULL REFERENCES teams (id) ON DELETE CASCADE,
    role       text   NOT NULL CHECK (role IN ('admin', 'operator', 'user')),
    CONSTRAINT group_mappings_key UNIQUE (provider, group_name, team_id)
);

-- Журнал аудита переживает удаление пользователей, команд и кластеров: user_id/team_id/cluster_id —
-- просто значения на момент события, без внешних ключей. username/role записываются как были.
CREATE TABLE audit_events (
    id         bigint GENERATED ALWAYS AS IDENTITY PRIMARY KEY,
    ts         timestamptz NOT NULL DEFAULT now(),
    username   text        NOT NULL DEFAULT '',
    role       text        NOT NULL DEFAULT '',
    action     text        NOT NULL,
    result     text        NOT NULL CHECK (result IN ('ok', 'failed', 'denied')),
    detail     text        NOT NULL DEFAULT '',
    source     text        NOT NULL DEFAULT '',
    user_id    text,
    team_id    text,
    cluster_id text
);
CREATE INDEX audit_events_ts_idx ON audit_events (ts DESC);
CREATE INDEX audit_events_team_ts_idx ON audit_events (team_id, ts DESC) WHERE team_id IS NOT NULL;
CREATE INDEX audit_events_cluster_ts_idx ON audit_events (cluster_id, ts DESC) WHERE cluster_id IS NOT NULL;

-- Общие настройки (например, время автовыхода): ключ → JSON.
CREATE TABLE settings (
    key        text        PRIMARY KEY,
    value      jsonb       NOT NULL,
    updated_at timestamptz NOT NULL DEFAULT now()
);

-- Заготовка под сессии в БД (этап 4, несколько реплик): в таблице только SHA-256 токена.
CREATE TABLE sessions (
    token_hash bytea       PRIMARY KEY,
    user_id    text        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    csrf       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    expires_at timestamptz NOT NULL
);
CREATE INDEX sessions_user_idx ON sessions (user_id);
CREATE INDEX sessions_expires_idx ON sessions (expires_at);
