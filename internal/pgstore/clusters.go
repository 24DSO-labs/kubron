// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package pgstore

import (
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Cluster — метаданные кластера в БД. Поля повторяют k8s.ClusterMeta, добавлены команда и режим подключения.
type Cluster struct {
	ID               string
	TeamID           string // при Save пусто → команда по умолчанию (у существующего кластера команда не меняется)
	DisplayName      string
	Icon             string
	Color            string
	Environment      string
	DefaultNamespace string
	ContextName      string
	Mode             string // kubeconfig | in-cluster | oidc-password; пусто → kubeconfig
	AddedAt          string // RFC3339; пусто → текущее время
}

func kubeconfigAAD(id string) []byte { return []byte("cluster/" + id + "/kubeconfig") }

// Clusters — хранилище кластеров с шифрованием kubeconfig.
func (s *Store) Clusters() *Clusters { return &Clusters{s} }

// Clusters — кластеры поверх таблицы clusters.
type Clusters struct{ s *Store }

const clusterCols = `id, team_id, display_name, icon, color, environment, default_namespace, context_name, mode, added_at`

func scanCluster(row pgx.Row) (*Cluster, error) {
	var c Cluster
	var added time.Time
	if err := row.Scan(&c.ID, &c.TeamID, &c.DisplayName, &c.Icon, &c.Color, &c.Environment, &c.DefaultNamespace,
		&c.ContextName, &c.Mode, &added); err != nil {
		return nil, err
	}
	c.AddedAt = added.UTC().Format(time.RFC3339)
	return &c, nil
}

// List возвращает все кластеры (фильтрация по команде — в слое авторизации, задача 3.2).
func (c *Clusters) List() ([]Cluster, error) {
	ctx, cancel := opCtx()
	defer cancel()
	rows, err := c.s.pool.Query(ctx, `SELECT `+clusterCols+` FROM clusters ORDER BY added_at, id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Cluster
	for rows.Next() {
		cl, err := scanCluster(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *cl)
	}
	return out, rows.Err()
}

// Get возвращает кластер или ErrClusterNotFound.
func (c *Clusters) Get(id string) (*Cluster, error) {
	ctx, cancel := opCtx()
	defer cancel()
	cl, err := scanCluster(c.s.pool.QueryRow(ctx, `SELECT `+clusterCols+` FROM clusters WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %q", ErrClusterNotFound, id)
	}
	return cl, err
}

// Kubeconfig расшифровывает kubeconfig кластера.
func (c *Clusters) Kubeconfig(id string) ([]byte, error) {
	ctx, cancel := opCtx()
	defer cancel()
	var ct []byte
	var keyID string
	err := c.s.pool.QueryRow(ctx, `SELECT kubeconfig_enc, key_id FROM clusters WHERE id = $1`, id).Scan(&ct, &keyID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, fmt.Errorf("%w: %q", ErrClusterNotFound, id)
	}
	if err != nil {
		return nil, err
	}
	if ct == nil { // in-cluster: kubeconfig'а нет
		return nil, nil
	}
	return c.s.keys.Open(ct, keyID, kubeconfigAAD(id))
}

// Save создаёт или обновляет кластер. kubeconfig == nil оставляет прежний; для нового кластера не в режиме
// in-cluster он обязателен. Пароль oidc-password (secret_enc) появится в задаче 3.7.
func (c *Clusters) Save(cl *Cluster, kubeconfig []byte) error {
	if err := c.s.ready(); err != nil {
		return err
	}
	mode := cl.Mode
	if mode == "" {
		mode = "kubeconfig"
	}
	team := cl.TeamID
	if team == "" {
		team = c.s.teamID
	}
	added := time.Now()
	if cl.AddedAt != "" {
		t, err := time.Parse(time.RFC3339, cl.AddedAt)
		if err != nil {
			return fmt.Errorf("invalid cluster date: %w", err)
		}
		added = t
	}
	var enc []byte
	keyID := ""
	if kubeconfig != nil {
		var err error
		if enc, keyID, err = c.s.keys.Seal(kubeconfig, kubeconfigAAD(cl.ID)); err != nil {
			return err
		}
	}
	ctx, cancel := opCtx()
	defer cancel()
	defer c.s.invalidateAccess()
	if kubeconfig == nil {
		// Обновление без смены kubeconfig: CHECK-ограничения проверяются до разрешения конфликта, поэтому
		// вставка строки без kubeconfig в ON CONFLICT не прошла бы даже для существующего кластера.
		tag, err := c.s.pool.Exec(ctx, `
			UPDATE clusters SET display_name = $2, icon = $3, color = $4, environment = $5, default_namespace = $6,
				context_name = $7, mode = $8, updated_at = now() WHERE id = $1`,
			cl.ID, cl.DisplayName, cl.Icon, cl.Color, cl.Environment, cl.DefaultNamespace, cl.ContextName, mode)
		if err != nil {
			return clusterErr(cl, err)
		}
		if tag.RowsAffected() > 0 {
			return nil
		}
	}
	_, err := c.s.pool.Exec(ctx, `
		INSERT INTO clusters (id, team_id, display_name, icon, color, environment, default_namespace, context_name, mode, kubeconfig_enc, key_id, added_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12)
		ON CONFLICT (id) DO UPDATE SET display_name = EXCLUDED.display_name, icon = EXCLUDED.icon, color = EXCLUDED.color,
			environment = EXCLUDED.environment, default_namespace = EXCLUDED.default_namespace,
			context_name = EXCLUDED.context_name, mode = EXCLUDED.mode, updated_at = now(),
			kubeconfig_enc = EXCLUDED.kubeconfig_enc, key_id = EXCLUDED.key_id`,
		cl.ID, team, cl.DisplayName, cl.Icon, cl.Color, cl.Environment, cl.DefaultNamespace, cl.ContextName, mode,
		enc, keyID, added)
	return clusterErr(cl, err)
}

func clusterErr(cl *Cluster, err error) error {
	switch {
	case err == nil:
		return nil
	case isUnique(err):
		return fmt.Errorf("%w: cluster %q", ErrConflict, cl.DisplayName)
	case pgCode(err) == "23514": // CHECK: например, нет kubeconfig у нового кластера
		return fmt.Errorf("invalid cluster data: %w", err)
	}
	return err
}

// Delete удаляет кластер вместе с токеном Argo CD; отсутствие кластера не ошибка.
func (c *Clusters) Delete(id string) error {
	ctx, cancel := opCtx()
	defer cancel()
	defer c.s.invalidateAccess()
	_, err := c.s.pool.Exec(ctx, `DELETE FROM clusters WHERE id = $1`, id)
	return err
}
