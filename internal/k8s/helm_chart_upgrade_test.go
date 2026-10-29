// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"
	"testing"
)

func TestParseHelmChartVersion(t *testing.T) {
	for in, want := range map[string]string{
		"1.2.3":         "1.2.3",
		"v1.2.3":        "1.2.3",
		" 2.0.0-rc.1 ":  "2.0.0-rc.1",
		"1.0.0+build.5": "1.0.0+build.5",
	} {
		got, err := parseHelmChartVersion(in)
		if err != nil || got.String() != want {
			t.Errorf("%q: got %v, %v (want %s)", in, got, err, want)
		}
	}
	// пусто, неполная версия и диапазоны не принимаются: «последняя» версия — не выбор пользователя
	for _, in := range []string{"", "  ", "1.2", "latest", "^1.2.0", ">=1.0.0", "1.2.3 || 2.0.0", strings.Repeat("1", 200)} {
		if _, err := parseHelmChartVersion(in); err == nil {
			t.Errorf("%q: expected an error", in)
		}
	}
}

func TestValidateHelmChartUpgradeRequest(t *testing.T) {
	ok := HelmChartUpgradeRequest{Namespace: "prod", Name: "web", Repo: "bitnami", Version: "1.2.3"}
	if err := validateHelmChartUpgradeRequest(&ok); err != nil {
		t.Fatalf("valid request: %v", err)
	}
	for name, mutate := range map[string]func(*HelmChartUpgradeRequest){
		"namespace": func(r *HelmChartUpgradeRequest) { r.Namespace = "Prod" },
		"release":   func(r *HelmChartUpgradeRequest) { r.Name = "" },
		"repo":      func(r *HelmChartUpgradeRequest) { r.Repo = "../etc" },
		"version":   func(r *HelmChartUpgradeRequest) { r.Version = "" },
	} {
		r := ok
		mutate(&r)
		if err := validateHelmChartUpgradeRequest(&r); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestChartUpgradeValidatesBeforeNetwork(t *testing.T) {
	m := NewManager()
	bad := HelmChartUpgradeRequest{Namespace: "prod", Name: "web", Repo: "bitnami", Version: "latest"}
	if _, err := m.PreviewHelmChartUpgrade(bad); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("preview with a non-exact version: %v", err)
	}
	if _, err := m.UpgradeHelmChart(bad); err == nil || !strings.Contains(err.Error(), "version") {
		t.Errorf("upgrade with a non-exact version: %v", err)
	}
	bad.Version = "1.0.0"
	bad.Values = "- not a map"
	if _, err := m.PreviewHelmChartUpgrade(bad); err == nil || !strings.Contains(err.Error(), "YAML") {
		t.Errorf("non-map values: %v", err)
	}
	if _, err := m.GetHelmUpgradeSources("prod", ""); err == nil {
		t.Error("an empty release name must be rejected")
	}
}

func TestFindHelmUpgradeSources(t *testing.T) {
	owner := idxWith("bitnami", "2.0.0", "1.5.0", "1.4.0", "1.3.0-rc.1")
	other := idxWith("mirror", "9.0.0", "8.0.0")
	empty := helmRepoIndex{name: "empty"}

	got := findHelmUpgradeSources([]helmRepoIndex{other, empty, owner}, "web", "1.4.0")
	if len(got) != 2 {
		t.Fatalf("expected 2 sources, got %+v", got)
	}
	// репозиторий, где есть установленная версия, идёт первым, остальные — по имени
	if got[0].Repo != "bitnami" || !got[0].Owns || got[1].Repo != "mirror" || got[1].Owns {
		t.Errorf("order/owns: %+v", got)
	}
	want := map[string]string{"2.0.0": "newer", "1.5.0": "newer", "1.4.0": "current", "1.3.0-rc.1": "older"}
	if len(got[0].Versions) != len(want) {
		t.Fatalf("versions: %+v", got[0].Versions)
	}
	for _, v := range got[0].Versions {
		if v.Relation != want[v.Version] {
			t.Errorf("%s: relation %q, want %q", v.Version, v.Relation, want[v.Version])
		}
		if v.Prerelease != (v.Version == "1.3.0-rc.1") {
			t.Errorf("%s: prerelease = %v", v.Version, v.Prerelease)
		}
	}
	if got[0].Versions[0].Version != "2.0.0" || got[0].Versions[3].Version != "1.3.0-rc.1" {
		t.Errorf("versions must go from newest to oldest: %+v", got[0].Versions)
	}
	// чужой репозиторий: сравнение с установленной версией всё равно есть
	if got[1].Versions[0].Relation != "newer" {
		t.Errorf("mirror 9.0.0: %+v", got[1].Versions[0])
	}
}

func TestFindHelmUpgradeSourcesEdgeCases(t *testing.T) {
	// индекс пришёл неотсортированным, версия с префиксом v нормализуется
	got := findHelmUpgradeSources([]helmRepoIndex{idxWith("r", "1.0.0", "v3.0.0", "2.0.0")}, "web", "2.0.0")
	if len(got) != 1 || got[0].Versions[0].Version != "3.0.0" || got[0].Versions[2].Version != "1.0.0" {
		t.Errorf("sorting: %+v", got)
	}
	// установленная версия не semver: источники есть, но связь с ней неизвестна
	got = findHelmUpgradeSources([]helmRepoIndex{idxWith("r", "2.0.0", "1.0.0")}, "web", "not-a-version")
	if len(got) != 1 || got[0].Owns || got[0].Versions[0].Relation != "" {
		t.Errorf("non-semver installed version: %+v", got)
	}
	// версии, которые нельзя выбрать (не strict semver), не предлагаются
	got = findHelmUpgradeSources([]helmRepoIndex{idxWith("r", "2.0.0", "1.2", "broken")}, "web", "2.0.0")
	if len(got) != 1 || len(got[0].Versions) != 1 {
		t.Errorf("non-strict versions: %+v", got)
	}
	// чарта нет ни в одном репозитории
	if got := findHelmUpgradeSources([]helmRepoIndex{idxWith("r", "1.0.0")}, "other", "1.0.0"); len(got) != 0 {
		t.Errorf("unknown chart: %+v", got)
	}
}
