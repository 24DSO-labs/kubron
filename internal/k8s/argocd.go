// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/discovery"
)

const (
	argoGroup        = "argoproj.io"
	argoGroupVersion = "argoproj.io/v1alpha1"
)

// argoInstalled проверяет через discovery, что в кластере зарегистрирован CRD applications.argoproj.io.
func argoInstalled(d discovery.DiscoveryInterface) (bool, error) {
	list, err := d.ServerResourcesForGroupVersion(argoGroupVersion)
	if err != nil {
		if apierrors.IsNotFound(err) {
			return false, nil
		}
		return false, fmt.Errorf("discovery %s: %w", argoGroupVersion, err)
	}
	for _, r := range list.APIResources {
		if r.Name == "applications" {
			return true, nil
		}
	}
	return false, nil
}

// ArgoInstalled сообщает, установлен ли Argo CD (CRD Application) в активном кластере.
func (m *Manager) ArgoInstalled() (bool, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return false, err
	}
	return argoInstalled(conn.DiscoveryClient)
}
