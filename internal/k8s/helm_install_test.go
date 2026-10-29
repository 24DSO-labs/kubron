// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"
)

func validInstallReq() HelmInstallRequest {
	return HelmInstallRequest{Repo: "bitnami", Chart: "nginx", Version: "1.0.0", ReleaseName: "web", Namespace: "prod"}
}

func TestValidateHelmInstallRequest(t *testing.T) {
	ok := validInstallReq()
	if err := validateHelmInstallRequest(&ok); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(r *HelmInstallRequest){
		"bad repository":     func(r *HelmInstallRequest) { r.Repo = "../x" },
		"empty chart":        func(r *HelmInstallRequest) { r.Chart = "" },
		"release name":       func(r *HelmInstallRequest) { r.ReleaseName = "Web_1" },
		"long release name":  func(r *HelmInstallRequest) { r.ReleaseName = strings.Repeat("a", 54) },
		"empty release name": func(r *HelmInstallRequest) { r.ReleaseName = "" },
		"namespace":          func(r *HelmInstallRequest) { r.Namespace = "Prod" },
		"empty namespace":    func(r *HelmInstallRequest) { r.Namespace = "" },
	}
	for name, mut := range cases {
		r := validInstallReq()
		mut(&r)
		if err := validateHelmInstallRequest(&r); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// Ошибки ввода должны возвращаться до сетевых вызовов и обращения к кластеру.
func TestInstallValidatesBeforeNetwork(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()

	r := validInstallReq()
	r.Values = "- not-a-map"
	if _, err := m.PreviewHelmInstall(r); err == nil || !strings.Contains(err.Error(), "YAML") {
		t.Errorf("non-map values: %v", err)
	}
	r = validInstallReq()
	r.ReleaseName = "BAD"
	if _, err := m.InstallHelmChart(r); err == nil {
		t.Error("bad release name")
	}
	// репозиторий не добавлен
	if _, err := m.PreviewHelmInstall(validInstallReq()); err == nil || !strings.Contains(err.Error(), "not found") {
		t.Errorf("nonexistent repository: %v", err)
	}
}
