-- Откат стирает секреты подключения: прежняя схема не умела их хранить отдельно от kubeconfig.
UPDATE clusters SET secret_enc = NULL;
ALTER TABLE clusters DROP CONSTRAINT IF EXISTS clusters_secret_key;
ALTER TABLE clusters DROP CONSTRAINT IF EXISTS clusters_kubeconfig_key;
ALTER TABLE clusters ADD CONSTRAINT clusters_cipher_key CHECK ((kubeconfig_enc IS NULL AND secret_enc IS NULL) OR key_id <> '');
ALTER TABLE clusters DROP COLUMN secret_key_id;
