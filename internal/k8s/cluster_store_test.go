// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"errors"
	"io/fs"
	"sort"
	"sync"
	"testing"
)

// memClusterStore — хранилище в памяти без файлов (как будущее серверное): проверяет, что ядро
// не зависит от пути к kubeconfig.
type memClusterStore struct {
	mu   sync.Mutex
	meta map[string]ClusterMeta
	kc   map[string][]byte
}

func newMemClusterStore() *memClusterStore {
	return &memClusterStore{meta: map[string]ClusterMeta{}, kc: map[string][]byte{}}
}

func (s *memClusterStore) List() ([]ClusterMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []ClusterMeta
	for id, m := range s.meta {
		if _, ok := s.kc[id]; ok {
			out = append(out, m)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out, nil
}

func (s *memClusterStore) Get(id string) (*ClusterMeta, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	m, ok := s.meta[id]
	if !ok {
		return nil, fs.ErrNotExist
	}
	return &m, nil
}

func (s *memClusterStore) Kubeconfig(id string) ([]byte, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	b, ok := s.kc[id]
	if !ok {
		return nil, errors.New("kubeconfig not found for cluster " + id)
	}
	return append([]byte(nil), b...), nil
}

func (s *memClusterStore) Save(meta *ClusterMeta, kubeconfig []byte) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.meta[meta.ID] = *meta
	if kubeconfig != nil {
		s.kc[meta.ID] = append([]byte(nil), kubeconfig...)
	}
	return nil
}

func (s *memClusterStore) Delete(id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.meta, id)
	delete(s.kc, id)
	return nil
}

// clusterStoreContract — общие требования ко всем реализациям ClusterStore (файловая, память, позже PostgreSQL).
func clusterStoreContract(t *testing.T, s ClusterStore) {
	t.Helper()
	if list, err := s.List(); err != nil || len(list) != 0 {
		t.Fatalf("empty store: %v %v", list, err)
	}
	if _, err := s.Get("nope"); err == nil {
		t.Error("Get of a nonexistent cluster must return an error")
	}
	if _, err := s.Kubeconfig("nope"); err == nil {
		t.Error("Kubeconfig of a nonexistent cluster must return an error")
	}

	meta := &ClusterMeta{ID: "c1", DisplayName: "One", KubeconfigFile: "c1.kubeconfig", ContextName: "dev"}
	if err := s.Save(meta, []byte("kc-1")); err != nil {
		t.Fatal(err)
	}
	got, err := s.Get("c1")
	if err != nil || got.DisplayName != "One" || got.ContextName != "dev" {
		t.Fatalf("Get: %+v %v", got, err)
	}
	if kc, err := s.Kubeconfig("c1"); err != nil || string(kc) != "kc-1" {
		t.Fatalf("Kubeconfig: %q %v", kc, err)
	}

	// Сохранение без kubeconfig меняет только метаданные.
	meta.DisplayName = "Uno"
	if err := s.Save(meta, nil); err != nil {
		t.Fatal(err)
	}
	if kc, _ := s.Kubeconfig("c1"); string(kc) != "kc-1" {
		t.Errorf("kubeconfig changed while saving metadata: %q", kc)
	}
	if got, _ := s.Get("c1"); got.DisplayName != "Uno" {
		t.Errorf("metadata was not updated: %+v", got)
	}

	// Замена kubeconfig.
	if err := s.Save(meta, []byte("kc-2")); err != nil {
		t.Fatal(err)
	}
	if kc, _ := s.Kubeconfig("c1"); string(kc) != "kc-2" {
		t.Errorf("kubeconfig was not replaced: %q", kc)
	}

	if list, _ := s.List(); len(list) != 1 || list[0].ID != "c1" {
		t.Errorf("List: %+v", list)
	}

	if err := s.Delete("c1"); err != nil {
		t.Fatal(err)
	}
	if err := s.Delete("c1"); err != nil {
		t.Errorf("deleting again must not be an error: %v", err)
	}
	if list, _ := s.List(); len(list) != 0 {
		t.Errorf("after deletion: %+v", list)
	}
}

func TestFileClusterStoreContract(t *testing.T) {
	setupClustersHome(t)
	clusterStoreContract(t, NewFileClusterStore())
}

func TestMemClusterStoreContract(t *testing.T) {
	clusterStoreContract(t, newMemClusterStore())
}

// useClusterStore подменяет хранилище на время теста.
func useClusterStore(t *testing.T, s ClusterStore) {
	t.Helper()
	prev := clusterStore
	clusterStore = s
	t.Cleanup(func() { clusterStore = prev })
}

// Весь жизненный цикл кластера работает на хранилище без файлов, а подключение строится из байтов.
func TestClusterLifecycleOnNonFileStore(t *testing.T) {
	setupClustersHome(t)
	useClusterStore(t, newMemClusterStore())

	meta, err := AddClusterFromText(sampleKubeconfig, ClusterSettings{DisplayName: "mem one", ContextName: "prod"})
	if err != nil {
		t.Fatal(err)
	}
	if p := GetClusterKubeconfigPath(meta.ID); p != "" {
		t.Errorf("a store without files has no path, got %q", p)
	}
	if list, _ := ListClusters(); len(list) != 1 || list[0].ID != meta.ID {
		t.Fatalf("ListClusters: %+v", list)
	}
	if text, err := ReadClusterKubeconfig(meta.ID); err != nil || text != sampleKubeconfig {
		t.Fatalf("ReadClusterKubeconfig: %v", err)
	}

	res, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "mem two", ContextName: "dev"}, KubeconfigSource{})
	if err != nil || !res.ConnectionChanged || res.Cluster.ContextName != "dev" {
		t.Fatalf("UpdateClusterSettings: %+v %v", res, err)
	}

	// Подключение строится из байтов kubeconfig, контекст берётся из метаданных.
	conn, err := buildClusterConnection(meta.ID)
	if err != nil {
		t.Fatal(err)
	}
	if conn.Config.Host != "https://10.0.0.1:6443" { // контекст dev → кластер c1
		t.Errorf("host %q: the context from metadata was not applied", conn.Config.Host)
	}

	if err := DeleteCluster(meta.ID); err != nil {
		t.Fatal(err)
	}
	if list, _ := ListClusters(); len(list) != 0 {
		t.Errorf("after deletion: %+v", list)
	}
}
