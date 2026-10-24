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
		t.Fatalf("корректные настройки отвергнуты: %v", err)
	}
	if got.DisplayName != "Prod EU" || got.Color != "#a6e3a1" || got.Environment != "prod" {
		t.Errorf("настройки не приведены к каноничному виду: %+v", got)
	}

	if got, err := (ClusterSettings{DisplayName: "x"}).normalize(""); err != nil || got.Color != defaultColor {
		t.Errorf("пустой цвет должен заменяться цветом по умолчанию: %+v %v", got, err)
	}
	if _, err := (ClusterSettings{DisplayName: "x", Icon: "🔵"}).normalize("🔵"); err != nil {
		t.Errorf("прежний значок-эмодзи должен сохраняться: %v", err)
	}

	bad := map[string]ClusterSettings{
		"пустое название":       {DisplayName: "   "},
		"длинное название":      {DisplayName: strings.Repeat("я", maxClusterNameLen+1)},
		"неизвестное окружение": {DisplayName: "x", Environment: "qa"},
		"цвет":      {DisplayName: "x", Color: "red"},
		"значок":    {DisplayName: "x", Icon: "🔵"},
		"namespace": {DisplayName: "x", DefaultNamespace: "Bad_NS"},
	}
	for name, s := range bad {
		if _, err := s.normalize(""); err == nil {
			t.Errorf("%s: ожидалась ошибка", name)
		}
	}
}

func TestPickContext(t *testing.T) {
	cfg, err := parseKubeconfig([]byte(sampleKubeconfig))
	if err != nil {
		t.Fatal(err)
	}
	if got, _ := pickContext(cfg, "prod"); got != "prod" {
		t.Errorf("явный выбор: %q", got)
	}
	if got, _ := pickContext(cfg, ""); got != "dev" {
		t.Errorf("current-context: %q", got)
	}
	if _, err := pickContext(cfg, "nope"); err == nil {
		t.Error("несуществующий контекст должен давать ошибку")
	}

	cfg.CurrentContext = ""
	if got, _ := pickContext(cfg, ""); got != "dev" {
		t.Errorf("без current-context берётся первый по алфавиту, получено %q", got)
	}
}

func TestInspectKubeconfig(t *testing.T) {
	info, err := InspectKubeconfig(KubeconfigSource{Text: sampleKubeconfig})
	if err != nil {
		t.Fatal(err)
	}
	if info.CurrentContext != "dev" || len(info.Contexts) != 2 {
		t.Fatalf("неожиданный результат: %+v", info)
	}
	prod := info.Contexts[1]
	if prod.Name != "prod" || prod.Server != "https://10.0.0.2:6443" || prod.Namespace != "apps" {
		t.Errorf("контекст prod описан неверно: %+v", prod)
	}

	if _, err := InspectKubeconfig(KubeconfigSource{}); err == nil {
		t.Error("пустой источник должен давать ошибку")
	}
	if _, err := InspectKubeconfig(KubeconfigSource{Text: sampleKubeconfig, Path: "/tmp/x"}); err == nil {
		t.Error("два источника сразу должны давать ошибку")
	}
	if _, err := InspectKubeconfig(KubeconfigSource{Text: "это не kubeconfig"}); err == nil {
		t.Error("мусор вместо kubeconfig должен давать ошибку")
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
		t.Fatalf("метаданные добавленного кластера: %+v", meta)
	}

	// Только оформление: подключение не меняется.
	res, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", Color: "#f38ba8", Environment: "stage"}, KubeconfigSource{})
	if err != nil {
		t.Fatal(err)
	}
	if res.ConnectionChanged {
		t.Error("смена названия и цвета не должна сбрасывать подключение")
	}
	if res.Cluster.DisplayName != "nemo stage" || res.Cluster.Environment != "stage" || res.Cluster.ContextName != "prod" {
		t.Errorf("после правки оформления: %+v", res.Cluster)
	}

	// Смена контекста — подключение устарело.
	res, err = UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", ContextName: "dev"}, KubeconfigSource{})
	if err != nil {
		t.Fatal(err)
	}
	if !res.ConnectionChanged || res.Cluster.ContextName != "dev" {
		t.Errorf("смена контекста: %+v", res)
	}

	// Неизвестный контекст отклоняется, сохранённое остаётся прежним.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage", ContextName: "nope"}, KubeconfigSource{}); err == nil {
		t.Error("неизвестный контекст должен давать ошибку")
	}
	if got, _ := LoadClusterMeta(meta.ID); got.ContextName != "dev" {
		t.Errorf("после отказа контекст изменился: %q", got.ContextName)
	}

	kcPath := GetClusterKubeconfigPath(meta.ID)
	before, err := os.ReadFile(kcPath)
	if err != nil {
		t.Fatal(err)
	}

	// Битый kubeconfig отклоняется до записи.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: "nemo stage"}, KubeconfigSource{Text: "мусор: ["}); err == nil {
		t.Error("битый kubeconfig должен отклоняться")
	}
	if after, _ := os.ReadFile(kcPath); string(after) != string(before) {
		t.Error("kubeconfig изменился, хотя замена была отклонена")
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
		t.Errorf("замена kubeconfig: %+v", res)
	}
	if after, _ := os.ReadFile(kcPath); string(after) != singleKubeconfig {
		t.Error("kubeconfig не заменён")
	}

	// Пустое название не принимается.
	if _, err := UpdateClusterSettings(meta.ID, ClusterSettings{DisplayName: " "}, KubeconfigSource{}); err == nil {
		t.Error("пустое название должно отклоняться")
	}

	// Временные файлы атомарной записи не остаются в каталоге.
	entries, err := os.ReadDir(getClustersDir())
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), ".tmp-") {
			t.Errorf("остался временный файл %s", e.Name())
		}
	}
}

func TestAddClusterRejectsBrokenKubeconfig(t *testing.T) {
	setupClustersHome(t)

	if _, err := AddClusterFromText("не kubeconfig", ClusterSettings{DisplayName: "bad"}); err == nil {
		t.Fatal("битый kubeconfig должен отклоняться при добавлении")
	}
	list, _ := ListClusters()
	if len(list) != 0 {
		t.Errorf("после отказа в списке остались кластеры: %+v", list)
	}
}

func TestClusterIDCannotEscapeDirectory(t *testing.T) {
	setupClustersHome(t)

	for _, id := range []string{"../x", "a/b", `a\b`, "", ".."} {
		if _, err := LoadClusterMeta(id); err == nil {
			t.Errorf("LoadClusterMeta(%q) должен отклонять идентификатор", id)
		}
		if err := DeleteCluster(id); err == nil {
			t.Errorf("DeleteCluster(%q) должен отклонять идентификатор", id)
		}
	}
}
