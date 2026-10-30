-- Секрет подключения кластера (пароль oidc-password, задача 3.7) шифруется отдельно от kubeconfig и
-- может быть зашифрован другим ключом после ротации: у каждого шифротекста свой идентификатор ключа.
ALTER TABLE clusters ADD COLUMN secret_key_id text NOT NULL DEFAULT '';
ALTER TABLE clusters DROP CONSTRAINT clusters_cipher_key;
ALTER TABLE clusters ADD CONSTRAINT clusters_kubeconfig_key CHECK (kubeconfig_enc IS NULL OR key_id <> '');
ALTER TABLE clusters ADD CONSTRAINT clusters_secret_key CHECK (secret_enc IS NULL OR secret_key_id <> '');
