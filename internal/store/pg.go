// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"errors"

	"github.com/VadimIl224/kubron/internal/audit"
	"github.com/VadimIl224/kubron/internal/auth"
	"github.com/VadimIl224/kubron/internal/k8s"
	"github.com/VadimIl224/kubron/internal/pgstore"
)

// FromPostgres адаптирует хранилище сервиса на PostgreSQL (internal/pgstore) к Store.
func FromPostgres(s *pgstore.Store) Store { return pgBackend{s} }

type pgBackend struct{ s *pgstore.Store }

func (p pgBackend) Users() auth.Backend        { return auth.FromExt(p.s.Users()) }
func (p pgBackend) Audit() audit.Backend       { return p.s.Audit() }
func (p pgBackend) ArgoTokens() k8s.TokenStore { return p.s.ArgoTokens() }

// Access — права по членству в командах (задача 3.2); app подключает их к сервису аутентификации.
func (p pgBackend) Access() auth.Access { return p.s.Access() }

// Directory — команды и участники (задача 3.3).
func (p pgBackend) Directory() auth.Directory  { return p.s.Directory() }
func (p pgBackend) Clusters() k8s.ClusterStore { return pgClusters{p.s.Clusters()} }

// pgClusters приводит pgstore.Clusters к k8s.ClusterStore. Файла kubeconfig у кластера в БД нет
// (kubeconfigPather не реализован): клиент получает байты.
type pgClusters struct{ c *pgstore.Clusters }

func (p pgClusters) List() ([]k8s.ClusterMeta, error) {
	list, err := p.c.List()
	if err != nil {
		return nil, err
	}
	out := make([]k8s.ClusterMeta, 0, len(list))
	for i := range list {
		out = append(out, toMeta(&list[i]))
	}
	return out, nil
}

func (p pgClusters) Get(id string) (*k8s.ClusterMeta, error) {
	c, err := p.c.Get(id)
	if err != nil {
		return nil, err
	}
	m := toMeta(c)
	return &m, nil
}

func (p pgClusters) Kubeconfig(id string) ([]byte, error) { return p.c.Kubeconfig(id) }

func (p pgClusters) Save(m *k8s.ClusterMeta, kubeconfig []byte) error {
	if m == nil {
		return errors.New("cluster is not set")
	}
	return p.c.Save(&pgstore.Cluster{
		ID: m.ID, DisplayName: m.DisplayName, Icon: m.Icon, Color: m.Color, Environment: m.Environment,
		DefaultNamespace: m.DefaultNamespace, ContextName: m.ContextName, AddedAt: m.AddedAt, TeamID: m.TeamID,
	}, kubeconfig)
}

func (p pgClusters) Delete(id string) error { return p.c.Delete(id) }

func toMeta(c *pgstore.Cluster) k8s.ClusterMeta {
	return k8s.ClusterMeta{
		ID: c.ID, DisplayName: c.DisplayName, Icon: c.Icon, Color: c.Color, Environment: c.Environment,
		DefaultNamespace: c.DefaultNamespace, ContextName: c.ContextName, AddedAt: c.AddedAt, TeamID: c.TeamID,
	}
}
