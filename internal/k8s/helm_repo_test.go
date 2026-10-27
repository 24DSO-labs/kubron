// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"runtime"
	"strings"
	"sync"
	"testing"
)

func setupHelmHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	old := helmHomeDir
	helmHomeDir = func() string { return dir }
	reset := func() {
		helmIndexMu.Lock()
		helmIndexCache = map[string]helmCachedIndex{}
		helmIndexMu.Unlock()
		helmPkgMu.Lock()
		helmPkgCache = map[string]*HelmChartPackage{}
		helmPkgOrder = nil
		helmPkgMu.Unlock()
	}
	reset()
	t.Cleanup(func() { helmHomeDir = old; reset() })
}

func makeChartTgz(t *testing.T, name, version string) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	files := map[string]string{
		name + "/Chart.yaml":  fmt.Sprintf("apiVersion: v2\nname: %s\nversion: %s\n", name, version),
		name + "/values.yaml": "replicas: 2\n",
		name + "/README.md":   "hello readme\n",
	}
	for n, body := range files {
		if err := tw.WriteHeader(&tar.Header{Name: n, Mode: 0o644, Size: int64(len(body))}); err != nil {
			t.Fatal(err)
		}
		if _, err := tw.Write([]byte(body)); err != nil {
			t.Fatal(err)
		}
	}
	if err := tw.Close(); err != nil {
		t.Fatal(err)
	}
	if err := gz.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func indexYAML(chartURL, digest string) string {
	return fmt.Sprintf(`apiVersion: v1
entries:
  web:
  - apiVersion: v2
    name: web
    version: 1.2.0
    appVersion: "2.0"
    description: Web server
    keywords: [nginx]
    urls: [%[1]s]
    digest: %[2]s
    created: "2024-01-01T00:00:00Z"
  - apiVersion: v2
    name: web
    version: 1.0.0
    description: Web server
    urls: [web-1.0.0.tgz]
    created: "2023-01-01T00:00:00Z"
`, chartURL, digest)
}

// authLog запоминает заголовок Authorization по путям.
type authLog struct {
	mu sync.Mutex
	m  map[string]string
}

func (a *authLog) set(path, v string) { a.mu.Lock(); a.m[path] = v; a.mu.Unlock() }
func (a *authLog) get(path string) string {
	a.mu.Lock()
	defer a.mu.Unlock()
	return a.m[path]
}

func newRepoServer(t *testing.T, tgz []byte, digest, chartURL, wantUser, wantPass string, log *authLog) *httptest.Server {
	t.Helper()
	h := http.NewServeMux()
	guard := func(w http.ResponseWriter, r *http.Request) bool {
		log.set(r.URL.Path, r.Header.Get("Authorization"))
		if wantUser != "" {
			u, p, ok := r.BasicAuth()
			if !ok || u != wantUser || p != wantPass {
				w.WriteHeader(http.StatusUnauthorized)
				return false
			}
		}
		return true
	}
	h.HandleFunc("/index.yaml", func(w http.ResponseWriter, r *http.Request) {
		if guard(w, r) {
			_, _ = w.Write([]byte(indexYAML(chartURL, digest)))
		}
	})
	h.HandleFunc("/web-1.2.0.tgz", func(w http.ResponseWriter, r *http.Request) {
		if guard(w, r) {
			_, _ = w.Write(tgz)
		}
	})
	srv := httptest.NewServer(h)
	t.Cleanup(srv.Close)
	return srv
}

func TestValidateHelmRepoName(t *testing.T) {
	for _, ok := range []string{"a", "bitnami", "my-repo_1.x", "0abc"} {
		if err := validateHelmRepoName(ok); err != nil {
			t.Errorf("%q: %v", ok, err)
		}
	}
	long := strings.Repeat("a", 64)
	for _, bad := range []string{"", "Bitnami", "../etc", "a/b", `a\b`, "-a", "a-", ".a", "a b", long} {
		if err := validateHelmRepoName(bad); err == nil {
			t.Errorf("%q должно отклоняться", bad)
		}
	}
}

func TestNormalizeHelmRepoURL(t *testing.T) {
	got, err := normalizeHelmRepoURL(" https://charts.example.com/stable/ ")
	if err != nil || got != "https://charts.example.com/stable" {
		t.Fatalf("got %q, %v", got, err)
	}
	for _, bad := range []string{
		"", "ftp://x/y", "file:///etc", "oci://ghcr.io/x", "https://", "https://u:p@x.io/",
		"https://x.io/?a=1", "https://x.io/#f", "charts.example.com",
	} {
		if _, err := normalizeHelmRepoURL(bad); err == nil {
			t.Errorf("%q должно отклоняться", bad)
		}
	}
}

func TestResolveChartURL(t *testing.T) {
	cases := map[string]string{
		"web-1.0.0.tgz":                  "https://r.io/stable/web-1.0.0.tgz",
		"/charts/web.tgz":                "https://r.io/charts/web.tgz",
		"https://cdn.io/web.tgz":         "https://cdn.io/web.tgz",
		"https://user:pw@cdn.io/web.tgz": "https://cdn.io/web.tgz",
	}
	for in, want := range cases {
		got, err := resolveChartURL("https://r.io/stable", in)
		if err != nil || got != want {
			t.Errorf("%q: got %q, %v (want %q)", in, got, err, want)
		}
	}
	for _, bad := range []string{"oci://x/y", "file:///etc/passwd", "ftp://x/y"} {
		if _, err := resolveChartURL("https://r.io/stable", bad); err == nil {
			t.Errorf("%q должно отклоняться", bad)
		}
	}
}

func TestHelmMatches(t *testing.T) {
	if !helmMatches(nil, "x", "", nil) {
		t.Error("пустой запрос находит всё")
	}
	if !helmMatches([]string{"web", "nginx"}, "web", "Server", []string{"NGINX"}) {
		t.Error("все слова должны находиться (регистр не важен)")
	}
	if helmMatches([]string{"web", "redis"}, "web", "Server", nil) {
		t.Error("отсутствующее слово должно отсекать")
	}
}

func TestHelmRepoLifecycle(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	tgz := makeChartTgz(t, "web", "1.2.0")
	sum := sha256.Sum256(tgz)
	log := &authLog{m: map[string]string{}}
	srv := newRepoServer(t, tgz, hex.EncodeToString(sum[:]), "web-1.2.0.tgz", "bob", "secret", log)

	// Некорректный ввод и неверный пароль не должны ничего сохранять
	if err := m.AddHelmRepo("Bad Name", srv.URL, "", "", false); err == nil {
		t.Error("плохое имя")
	}
	if err := m.AddHelmRepo("demo", "ftp://x", "", "", false); err == nil {
		t.Error("плохая схема")
	}
	if err := m.AddHelmRepo("demo", srv.URL, "bob", "wrong", false); err == nil {
		t.Error("неверный пароль должен давать ошибку")
	}
	if err := m.AddHelmRepo("demo", srv.URL, "", "pw-without-user", false); err == nil {
		t.Error("пароль без пользователя")
	}
	if repos, _ := m.ListHelmRepos(); len(repos) != 0 {
		t.Fatalf("после неудачных добавлений список должен быть пуст: %v", repos)
	}

	if err := m.AddHelmRepo("demo", srv.URL, "bob", "secret", false); err != nil {
		t.Fatal(err)
	}
	if err := m.AddHelmRepo("demo", srv.URL, "bob", "secret", false); err == nil {
		t.Error("повторное добавление должно давать ошибку")
	}
	if runtime.GOOS != "windows" {
		st, err := os.Stat(helmRepoFilePath())
		if err != nil || st.Mode().Perm() != 0o600 {
			t.Errorf("repositories.yaml: %v, %v", st, err)
		}
	}

	repos, err := m.ListHelmRepos()
	if err != nil || len(repos) != 1 {
		t.Fatalf("%v, %v", repos, err)
	}
	if r := repos[0]; r.Name != "demo" || r.Charts != 1 || r.Updated == "" || r.Error != "" || r.Username != "bob" {
		t.Errorf("repo: %+v", r)
	}

	res, err := m.SearchHelmCharts("nginx", "")
	if err != nil || res.Total != 1 || res.Items[0].Version != "1.2.0" || res.Items[0].Versions != 2 {
		t.Fatalf("search: %+v, %v", res, err)
	}
	if res, _ := m.SearchHelmCharts("zzz-nothing", ""); res.Total != 0 || res.Items == nil {
		t.Errorf("пустой результат: %+v", res)
	}
	if res, _ := m.SearchHelmCharts("", "other"); res.Total != 0 {
		t.Errorf("фильтр по несуществующему репозиторию: %+v", res)
	}

	vers, err := m.GetHelmChartVersions("demo", "web")
	if err != nil || len(vers) != 2 || vers[0].Version != "1.2.0" {
		t.Fatalf("versions: %+v, %v", vers, err)
	}
	if _, err := m.GetHelmChartVersions("demo", "nope"); err == nil {
		t.Error("несуществующий чарт")
	}

	pkg, err := m.GetHelmChartPackage("demo", "web", "1.2.0")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(pkg.Readme, "hello readme") || !strings.Contains(pkg.Values, "replicas: 2") {
		t.Errorf("pkg: %+v", pkg)
	}

	upd, err := m.UpdateHelmRepos("")
	if err != nil || len(upd) != 1 || upd[0].Error != "" || upd[0].Charts != 1 {
		t.Fatalf("update: %+v, %v", upd, err)
	}
	if _, err := m.UpdateHelmRepos("missing"); err == nil {
		t.Error("обновление несуществующего репозитория")
	}

	if err := m.RemoveHelmRepo("demo"); err != nil {
		t.Fatal(err)
	}
	if err := m.RemoveHelmRepo("demo"); err == nil {
		t.Error("повторное удаление")
	}
	if _, err := os.Stat(helmIndexPath("demo")); !os.IsNotExist(err) {
		t.Errorf("индекс должен быть удалён: %v", err)
	}
	if repos, _ := m.ListHelmRepos(); len(repos) != 0 {
		t.Errorf("список после удаления: %v", repos)
	}
}

func TestHelmChartDigestMismatch(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	tgz := makeChartTgz(t, "web", "1.2.0")
	log := &authLog{m: map[string]string{}}
	srv := newRepoServer(t, tgz, strings.Repeat("0", 64), "web-1.2.0.tgz", "", "", log)

	if err := m.AddHelmRepo("demo", srv.URL, "", "", false); err != nil {
		t.Fatal(err)
	}
	if _, err := m.GetHelmChartPackage("demo", "web", "1.2.0"); err == nil || !strings.Contains(err.Error(), "контрольная сумма") {
		t.Errorf("ожидалась ошибка контрольной суммы: %v", err)
	}
}

// Пароль уходит только на хост репозитория, но не на хост из абсолютной ссылки в index.yaml.
func TestHelmCredentialsNotSentToForeignHost(t *testing.T) {
	setupHelmHome(t)
	m := NewManager()
	tgz := makeChartTgz(t, "web", "1.2.0")
	sum := sha256.Sum256(tgz)
	digest := hex.EncodeToString(sum[:])

	foreignLog := &authLog{m: map[string]string{}}
	foreign := newRepoServer(t, tgz, digest, "web-1.2.0.tgz", "", "", foreignLog)

	repoLog := &authLog{m: map[string]string{}}
	repoSrv := newRepoServer(t, tgz, digest, foreign.URL+"/web-1.2.0.tgz", "bob", "secret", repoLog)

	if err := m.AddHelmRepo("demo", repoSrv.URL, "bob", "secret", false); err != nil {
		t.Fatal(err)
	}
	if repoLog.get("/index.yaml") == "" {
		t.Error("на хост репозитория учётные данные должны уходить")
	}
	if _, err := m.GetHelmChartPackage("demo", "web", "1.2.0"); err != nil {
		t.Fatal(err)
	}
	if got := foreignLog.get("/web-1.2.0.tgz"); got != "" {
		t.Errorf("на чужой хост утёк Authorization: %q", got)
	}
}

func TestHelmIndexSizeLimit(t *testing.T) {
	setupHelmHome(t)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		chunk := bytes.Repeat([]byte("#"), 1<<20)
		for i := 0; i <= helmIndexMaxBytes>>20; i++ {
			if _, err := w.Write(chunk); err != nil {
				return
			}
		}
	}))
	defer srv.Close()
	if err := NewManager().AddHelmRepo("big", srv.URL, "", "", false); err == nil || !strings.Contains(err.Error(), "больше") {
		t.Errorf("ожидалась ошибка лимита размера: %v", err)
	}
}
