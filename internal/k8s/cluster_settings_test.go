// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const sampleKubeconfig = `apiVersion: v1
kind: Config
current-context: dev
clusters:
- name: c1
  cluster:
    server: https://10.0.0.1:6443
- name: c2
  cluster:
    server: https://10.0.0.2:6443
contexts:
- name: dev
  context:
    cluster: c1
    user: u
- name: prod
  context:
    cluster: c2
    user: u
    namespace: apps
users:
- name: u
  user:
    token: abc
`

const singleKubeconfig = `apiVersion: v1
kind: Config
current-context: only
clusters:
- name: c3
  cluster:
    server: https://10.0.0.3:6443
contexts:
- name: only
  context:
    cluster: c3
    user: u
users:
- name: u
  user:
    token: def
`

func setupClustersHome(t *testing.T) {
	t.Helper()
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // Windows
	if err := InitKubronDir(); err != nil {
		t.Fatal(err)
	}
}

func TestNormalizeSettings(t *testing.T) {
	ok := ClusterSettings{DisplayName: "  Prod EU  ", Color: "#A6E3A1", Environment: "PROD", Icon: "server", DefaultNamespace: "apps"}
	got, err := ok.normalize("")
	if err != nil {
		t.Fatalf("valid settings were rejected: %v", err)
	}
	if got.DisplayName != "Prod EU" || got.Color != "#a6e3a1" || got.Environment != "prod" {
		t.Errorf("settings were not normalized: %+v", got)
	}

	if got, err := (ClusterSettings{DisplayName: "x"}).normalize(""); err != nil || got.Color != defaultColor {
		t.Errorf("an empty color must be replaced with the default: %+v %v", got, err)
	}
	if _, err := (ClusterSettings{DisplayName: "x", Icon: "🔵"}).normalize("🔵"); err != nil {
		t.Errorf("a previous emoji icon must be preserved: %v", err)
	}

	bad := map[string]ClusterSettings{
		"empty name":          {DisplayName: "   "},
		"long name":           {DisplayName: strings.Repeat("я", maxClusterNameLen+1)},
		"unknown environment": {DisplayName: "x", Environment: "qa"},
		"color":               {DisplayName: "x", Color: "red"},
		"icon":                {DisplayName: "x", Icon: "🔵"},
		"namespace":           {DisplayName: "x", DefaultNamespace: "Bad_NS"},
	}
	for name, s := range bad {
		if _, err := s.normalize(""); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

func TestPickContext(t *testing.T) {
	cfg, err := parseKubeconfig([]byte(sampleKubeconfig))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := pickContext(cfg, "prod"); got != "prod" {
		t.Errorf("explicit choice: %q", got)
	}
	if got, _ := pickContext(cfg, ""); got != "dev" {
		t.Errorf("current-context: %q", got)
	}
	if _, err := pickContext(cfg, "nope"); err == nil {
		t.Error("a nonexistent context must give an error")
	}

	cfg.CurrentContext = ""
	if got, _ := pickContext(cfg, ""); got != "dev" {
		t.Errorf("without current-context the first one alphabetically is used, got %q", got)
	}
}

func TestInspectKubeconfig(t *testing.T) {
	info, err := InspectKubeconfig(KubeconfigSource{Text: sampleKubeconfig})
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentContext != "dev" || len(info.Contexts) != 2 {
		t.Fatalf("unexpected result: %+v", info)
	}
	prod := info.Contexts[1]
	if prod.Name != "prod" || prod.Server != "https://10.0.0.2:6443" || prod.Namespace != "apps" {
		t.Errorf("context prod is described incorrectly: %+v", prod)
	}

	if _, err := InspectKubeconfig(KubeconfigSource{}); err == nil {
		t.Error("an empty source must give an error")
	}
	if _, err := InspectKubeconfig(KubeconfigSource{Text: sampleKubeconfig, Path: "/tmp/x"}); err == nil {
		t.Error("two sources at once must give an error")
	}
	if _, err := InspectKubeconfig(KubeconfigSource{Text: "this is not a kubeconfig"}); err == nil {
		t.Error("garbage instead of kubeconfig must give an error")
	}
}

func TestAddAndUpdateCluster(t *testing.T) {
	setupClustersHome(t)

	meta, err := AddClusterFromText(sampleKubeconfig, ClusterSettings{
		DisplayName: "nemo dev", Icon: "server", Color: "#89b4fa", Environment: "dev", ContextName: "prod",
	})
	if err != nil {
		t.Fatal(err)
	}
	if meta.ContextName != "prod" || meta.Environment != "dev" || meta.Icon != "server" {
		t.Fatalf("metadata of the added cluster: %+v", meta)
	}

	// Только оформление: подключение не меняется.
	res, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", Color: "#f38ba8", Environment: "stage"}, KubeconfigSource{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ConnectionChanged {
		t.Error("changing the name and color must not reset the connection")
	}
	if res.Cluster.DisplayName != "nemo stage" || res.Cluster.Environment != "stage" || res.Cluster.ContextName != "prod" {
		t.Errorf("after editing the appearance: %+v", res.Cluster)
	}

	// Смена контекста — подключение устарело.
	res, err = UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", ContextName: "dev"}, KubeconfigSource{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.ConnectionChanged || res.Cluster.ContextName != "dev" {
		t.Errorf("context change: %+v", res)
	}

	// Неизвестный контекст отклоняется, сохранённое остаётся прежним.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", ContextName: "nope"}, KubeconfigSource{}); err == nil {
		t.Error("an unknown context must give an error")
	}
	if got, _ := LoadClusterMeta(meta.ID); got.ContextName != "dev" {
		t.Errorf("context changed after rejection: %q", got.ContextName)
	}

	kcPath := GetClusterKubeconfigPath(meta.ID)
	before, err := os.ReadFile(kcPath)
	if err != nil {
		t.Fatal(err)
	}

	// Битый kubeconfig отклоняется до записи.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage"}, KubeconfigSource{Text: "garbage: ["}); err == nil {
		t.Error("a broken kubeconfig must be rejected")
	}
	if after, _ := os.ReadFile(kcPath); string(after) != string(before) {
		t.Error("kubeconfig changed although the replacement was rejected")
	}

	// Замена файлом: прежнего контекста в новом файле нет — берётся контекст по умолчанию.
	src := filepath.Join(t.TempDir(), "new.kubeconfig")
	if err := os.WriteFile(src, []byte(singleKubeconfig), 0o600); err != nil {
		t.Fatal(err)
	}
	res, err = UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage"}, KubeconfigSource{Path: src})
	if err != nil {
		t.Fatal(err)
	}
	if !res.ConnectionChanged || res.Cluster.ContextName != "only" {
		t.Errorf("kubeconfig replacement: %+v", res)
	}
	if after, _ := os.ReadFile(kcPath); string(after) != singleKubeconfig {
		t.Error("kubeconfig was not replaced")
	}

	// Пустое название не принимается.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: " "}, KubeconfigSource{}); err == nil {
		t.Error("an empty name must be rejected")
	}

	// Временные файлы атомарной записи не остаются в каталоге.
	entries, err := os.ReadDir(getClustersDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("a temporary file %s was left behind", e.Name())
		}
	}
}

func TestAddClusterRejectsBrokenKubeconfig(t *testing.T) {
	setupClustersHome(t)

	if _, err := AddClusterFromText("not a kubeconfig", ClusterSettings{DisplayName: "bad"}); err == nil {
		t.Fatal("a broken kubeconfig must be rejected on add")
	}
	list, _ := ListClusters()
	if len(list) != 0 {
		t.Errorf("clusters remained in the list after rejection: %+v", list)
	}
}

func TestClusterIDCannotEscapeDirectory(t *testing.T) {
	setupClustersHome(t)

	for _, id := range []string{"../x", "a/b", `a\b`, "", ".."} {
		if _, err := LoadClusterMeta(id); err == nil {
			t.Errorf("LoadClusterMeta(%q) must reject the ID", id)
		}
		if err := DeleteCluster(id); err == nil {
			t.Errorf("DeleteCluster(%q) must reject the ID", id)
		}
	}
}
