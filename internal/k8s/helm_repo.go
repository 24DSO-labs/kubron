// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"bytes"
	"context"
	"crypto/sha256"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/chart/v2/loader"
	repo "helm.sh/helm/v4/pkg/repo/v1"
	"sigs.k8s.io/yaml"
)

// Репозитории чартов (H3). Хранилище — ~/.kubron/helm/ (отдельно от ~/.config/helm пользователя):
//
//	repositories.yaml  — формат Helm (repo.File), права 0600: в нём лежит пароль, как и у helm CLI;
//	cache/<имя>-index.yaml — скачанные index.yaml.
//
// БЕЗОПАСНОСТЬ: имя репозитория входит в имя файла, поэтому проверяется строго (validateHelmRepoName).
// Учётные данные уходят только на хост репозитория (не на хост, куда ведёт ссылка из index.yaml
// или редирект). Размеры index.yaml и архива чарта ограничены.
const (
	helmIndexMaxBytes    = 64 << 20
	helmChartMaxBytes    = 32 << 20
	helmRepoHTTPTimeout  = 30 * time.Second
	helmSearchMaxResults = 500
	helmPackageTextMax   = 1 << 20
	helmUpdateParallel   = 4
	helmPackageCacheSize = 16
)

// helmHomeDir — каталог данных Helm в Kubron; переменная нужна тестам.
var helmHomeDir = func() string { return filepath.Join(GetKubronDir(), "helm") }

var (
	helmRepoMu     sync.Mutex // сериализует read-modify-write repositories.yaml и подмену index-файлов
	helmRepoNameRe = regexp.MustCompile(`^[a-z0-9]([a-z0-9_.-]{0,61}[a-z0-9])?$`)

	helmIndexMu    sync.Mutex
	helmIndexCache = map[string]helmCachedIndex{}

	helmPkgMu    sync.Mutex
	helmPkgCache = map[string]*HelmChartPackage{}
	helmPkgOrder []string
)

type helmCachedIndex struct {
	mod time.Time
	idx *repo.IndexFile
}

// ─── типы для UI ───

type HelmRepoInfo struct {
	Name        string `json:"name"`
	URL         string `json:"url"`
	Username    string `json:"username"`
	InsecureTLS bool   `json:"insecureTLS"`
	Charts      int    `json:"charts"`
	Updated     string `json:"updated"` // RFC3339; "" — индекс ещё не скачан
	Error       string `json:"error"`   // почему индекс не читается
}

type HelmRepoUpdateResult struct {
	Name   string `json:"name"`
	Charts int    `json:"charts"`
	Error  string `json:"error"`
}

type HelmCatalogChart struct {
	Repo        string `json:"repo"`
	Name        string `json:"name"`
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion"`
	Description string `json:"description"`
	Deprecated  bool   `json:"deprecated"`
	Versions    int    `json:"versions"`
}

type HelmCatalogResult struct {
	Items []HelmCatalogChart `json:"items"`
	Total int                `json:"total"` // найдено всего (Items может быть обрезан)
}

type HelmChartVersionInfo struct {
	Version     string `json:"version"`
	AppVersion  string `json:"appVersion"`
	Created     string `json:"created"`
	Description string `json:"description"`
	Deprecated  bool   `json:"deprecated"`
}

type HelmChartPackage struct {
	Repo        string   `json:"repo"`
	Name        string   `json:"name"`
	Version     string   `json:"version"`
	AppVersion  string   `json:"appVersion"`
	Description string   `json:"description"`
	Home        string   `json:"home"`
	Sources     []string `json:"sources"`
	Maintainers []string `json:"maintainers"`
	Keywords    []string `json:"keywords"`
	Deprecated  bool     `json:"deprecated"`
	Readme      string   `json:"readme"`
	Values      string   `json:"values"`
}

// ─── валидация ───

func validateHelmRepoName(name string) error {
	if !helmRepoNameRe.MatchString(name) {
		return fmt.Errorf("некорректное имя репозитория %q: нужны строчные латинские буквы, цифры, «-», «_», «.» (до 63 символов)", name)
	}
	return nil
}

// normalizeHelmRepoURL допускает только http(s) без логина в адресе, query и fragment.
func normalizeHelmRepoURL(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if strings.HasPrefix(strings.ToLower(raw), "oci://") {
		return "", errors.New("OCI-реестры пока не поддерживаются (будут в следующем шаге)")
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("некорректный URL: %w", err)
	}
	if u.Scheme != "http" && u.Scheme != "https" {
		return "", errors.New("URL репозитория должен начинаться с http:// или https://")
	}
	if u.Host == "" || u.Hostname() == "" {
		return "", errors.New("в URL не указан хост")
	}
	if u.User != nil {
		return "", errors.New("логин и пароль задаются отдельными полями, а не в URL")
	}
	if u.RawQuery != "" || u.Fragment != "" {
		return "", errors.New("URL не должен содержать query и fragment")
	}
	return strings.TrimRight(u.String(), "/"), nil
}

// ─── файлы ───

func helmRepoFilePath() string { return filepath.Join(helmHomeDir(), "repositories.yaml") }
func helmCacheDir() string     { return filepath.Join(helmHomeDir(), "cache") }

// helmIndexPath вызывается только с проверенным именем.
func helmIndexPath(name string) string {
	return filepath.Join(helmCacheDir(), name+"-index.yaml")
}

// loadHelmRepoFile читает repositories.yaml; отсутствие файла — пустой список.
func loadHelmRepoFile() (*repo.File, error) {
	b, err := os.ReadFile(helmRepoFilePath())
	if errors.Is(err, os.ErrNotExist) {
		return repo.NewFile(), nil
	}
	if err != nil {
		return nil, fmt.Errorf("не удалось прочитать список репозиториев: %w", err)
	}
	f := repo.NewFile()
	if err := yaml.Unmarshal(b, f); err != nil {
		return nil, fmt.Errorf("repositories.yaml повреждён: %w", err)
	}
	return f, nil
}

func saveHelmRepoFile(f *repo.File) error {
	f.Generated = time.Now()
	b, err := yaml.Marshal(f)
	if err != nil {
		return err
	}
	// writeFileAtomic (cluster_settings.go) каталог не создаёт
	if err := os.MkdirAll(helmHomeDir(), 0o700); err != nil {
		return err
	}
	return writeFileAtomic(helmRepoFilePath(), b, 0o600)
}

// ─── сеть ───

func sameHost(a, b string) bool {
	ua, err1 := url.Parse(a)
	ub, err2 := url.Parse(b)
	return err1 == nil && err2 == nil && strings.EqualFold(ua.Host, ub.Host) && ua.Scheme == ub.Scheme
}

// helmGet скачивает URL с ограничением размера. Basic-авторизация отправляется только на хост репозитория;
// Go сам убирает заголовок Authorization при редиректе на другой домен.
func helmGet(ctx context.Context, e *repo.Entry, rawURL string, limit int64) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, helmRepoHTTPTimeout)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, rawURL, nil)
	if err != nil {
		return nil, err
	}
	if e.Username != "" && sameHost(rawURL, e.URL) {
		req.SetBasicAuth(e.Username, e.Password)
	}
	tr := http.DefaultTransport.(*http.Transport).Clone()
	if e.InsecureSkipTLSVerify {
		tr.TLSClientConfig = &tls.Config{InsecureSkipVerify: true} //nolint:gosec // явный выбор пользователя в настройках репозитория
	}
	defer tr.CloseIdleConnections()
	client := &http.Client{
		Transport: tr,
		Timeout:   helmRepoHTTPTimeout,
		CheckRedirect: func(r *http.Request, via []*http.Request) error {
			if len(via) >= 10 {
				return errors.New("слишком много редиректов")
			}
			// https -> http отправил бы пароль открытым текстом
			if r.URL.Scheme == "http" && via[len(via)-1].URL.Scheme == "https" {
				return errors.New("редирект с https на http запрещён")
			}
			return nil
		},
	}

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("запрос к %s: %w", redactURLUser(rawURL), err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("%s: HTTP %d", redactURLUser(rawURL), resp.StatusCode)
	}
	data, err := io.ReadAll(io.LimitReader(resp.Body, limit+1))
	if err != nil {
		return nil, fmt.Errorf("чтение ответа %s: %w", redactURLUser(rawURL), err)
	}
	if int64(len(data)) > limit {
		return nil, fmt.Errorf("ответ %s больше %d МиБ", redactURLUser(rawURL), limit>>20)
	}
	return data, nil
}

func redactURLUser(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return "(некорректный URL)"
	}
	u.User = nil
	return u.String()
}

// downloadHelmIndex скачивает index.yaml во временный файл и проверяет его разбором.
// Подмена рабочего индекса — отдельным шагом (commitHelmIndex), под мьютексом.
func downloadHelmIndex(ctx context.Context, e *repo.Entry) (tmpPath string, idx *repo.IndexFile, err error) {
	data, err := helmGet(ctx, e, e.URL+"/index.yaml", helmIndexMaxBytes)
	if err != nil {
		return "", nil, err
	}
	if err := os.MkdirAll(helmCacheDir(), 0o700); err != nil {
		return "", nil, err
	}
	tmp, err := os.CreateTemp(helmCacheDir(), ".idx-*")
	if err != nil {
		return "", nil, err
	}
	tmpPath = tmp.Name()
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil {
		_ = os.Remove(tmpPath)
		return "", nil, errors.Join(werr, cerr)
	}
	idx, err = repo.LoadIndexFile(tmpPath)
	if err != nil {
		_ = os.Remove(tmpPath)
		return "", nil, fmt.Errorf("index.yaml не является индексом Helm-репозитория: %w", err)
	}
	return tmpPath, idx, nil
}

// commitHelmIndex подменяет рабочий индекс; вызывать под helmRepoMu.
func commitHelmIndex(name, tmpPath string) error {
	if err := os.Rename(tmpPath, helmIndexPath(name)); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	helmIndexMu.Lock()
	delete(helmIndexCache, name)
	helmIndexMu.Unlock()
	return nil
}

// loadHelmIndex отдаёт индекс репозитория из памяти (по mtime файла) или с диска.
func loadHelmIndex(name string) (*repo.IndexFile, time.Time, error) {
	path := helmIndexPath(name)
	st, err := os.Stat(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	helmIndexMu.Lock()
	c, ok := helmIndexCache[name]
	helmIndexMu.Unlock()
	if ok && c.mod.Equal(st.ModTime()) {
		return c.idx, c.mod, nil
	}
	idx, err := repo.LoadIndexFile(path)
	if err != nil {
		return nil, time.Time{}, err
	}
	helmIndexMu.Lock()
	helmIndexCache[name] = helmCachedIndex{mod: st.ModTime(), idx: idx}
	helmIndexMu.Unlock()
	return idx, st.ModTime(), nil
}

func countHelmCharts(idx *repo.IndexFile) int {
	n := 0
	for _, v := range idx.Entries {
		if len(v) > 0 {
			n++
		}
	}
	return n
}

// ─── управление репозиториями ───

func (m *Manager) ListHelmRepos() ([]HelmRepoInfo, error) {
	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err != nil {
		return nil, err
	}
	out := make([]HelmRepoInfo, 0, len(f.Repositories))
	for _, e := range f.Repositories {
		if e == nil || validateHelmRepoName(e.Name) != nil {
			continue
		}
		info := HelmRepoInfo{Name: e.Name, URL: e.URL, Username: e.Username, InsecureTLS: e.InsecureSkipTLSVerify}
		idx, mod, err := loadHelmIndex(e.Name)
		switch {
		case err == nil:
			info.Charts = countHelmCharts(idx)
			info.Updated = mod.UTC().Format(time.RFC3339)
		case errors.Is(err, os.ErrNotExist):
			info.Error = "индекс не скачан: выполните Update"
		default:
			info.Error = err.Error()
		}
		out = append(out, info)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out, nil
}

// AddHelmRepo добавляет репозиторий и сразу скачивает индекс: при ошибке ничего не сохраняется.
func (m *Manager) AddHelmRepo(name, rawURL, username, password string, insecureTLS bool) error {
	if err := validateHelmRepoName(name); err != nil {
		return err
	}
	u, err := normalizeHelmRepoURL(rawURL)
	if err != nil {
		return err
	}
	if password != "" && username == "" {
		return errors.New("пароль без имени пользователя")
	}
	e := &repo.Entry{Name: name, URL: u, Username: username, Password: password, InsecureSkipTLSVerify: insecureTLS}

	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err != nil {
		return err
	}
	if f.Has(name) {
		return fmt.Errorf("репозиторий %q уже добавлен", name)
	}

	tmp, _, err := downloadHelmIndex(context.Background(), e)
	if err != nil {
		return fmt.Errorf("репозиторий %q недоступен: %w", name, err)
	}

	helmRepoMu.Lock()
	defer helmRepoMu.Unlock()
	f, err = loadHelmRepoFile() // между проверкой и записью список мог измениться
	if err != nil {
		_ = os.Remove(tmp)
		return err
	}
	if f.Has(name) {
		_ = os.Remove(tmp)
		return fmt.Errorf("репозиторий %q уже добавлен", name)
	}
	if err := commitHelmIndex(name, tmp); err != nil {
		return err
	}
	f.Add(e)
	if err := saveHelmRepoFile(f); err != nil {
		_ = os.Remove(helmIndexPath(name))
		return fmt.Errorf("не удалось сохранить список репозиториев: %w", err)
	}
	return nil
}

func (m *Manager) RemoveHelmRepo(name string) error {
	if err := validateHelmRepoName(name); err != nil {
		return err
	}
	helmRepoMu.Lock()
	defer helmRepoMu.Unlock()
	f, err := loadHelmRepoFile()
	if err != nil {
		return err
	}
	if !f.Remove(name) {
		return fmt.Errorf("репозиторий %q не найден", name)
	}
	if err := saveHelmRepoFile(f); err != nil {
		return err
	}
	_ = os.Remove(helmIndexPath(name))
	helmIndexMu.Lock()
	delete(helmIndexCache, name)
	helmIndexMu.Unlock()
	return nil
}

// UpdateHelmRepos обновляет индексы: name == "" — все репозитории (до 4 параллельно).
// Ошибка одного репозитория не мешает остальным и возвращается в его результате.
func (m *Manager) UpdateHelmRepos(name string) ([]HelmRepoUpdateResult, error) {
	if name != "" {
		if err := validateHelmRepoName(name); err != nil {
			return nil, err
		}
	}
	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err != nil {
		return nil, err
	}
	var targets []*repo.Entry
	for _, e := range f.Repositories {
		if e != nil && validateHelmRepoName(e.Name) == nil && (name == "" || e.Name == name) {
			targets = append(targets, e)
		}
	}
	if name != "" && len(targets) == 0 {
		return nil, fmt.Errorf("репозиторий %q не найден", name)
	}

	results := make([]HelmRepoUpdateResult, len(targets))
	sem := make(chan struct{}, helmUpdateParallel)
	var wg sync.WaitGroup
	for i, e := range targets {
		wg.Add(1)
		go func(i int, e *repo.Entry) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			results[i] = updateOneHelmRepo(e)
		}(i, e)
	}
	wg.Wait()
	sort.Slice(results, func(i, j int) bool { return results[i].Name < results[j].Name })
	return results, nil
}

func updateOneHelmRepo(e *repo.Entry) HelmRepoUpdateResult {
	res := HelmRepoUpdateResult{Name: e.Name}
	tmp, idx, err := downloadHelmIndex(context.Background(), e)
	if err != nil {
		res.Error = err.Error()
		return res
	}
	helmRepoMu.Lock()
	defer helmRepoMu.Unlock()
	// репозиторий могли удалить или переназначить, пока шла загрузка
	cur, err := loadHelmRepoFile()
	if err != nil || cur.Get(e.Name) == nil || cur.Get(e.Name).URL != e.URL {
		_ = os.Remove(tmp)
		res.Error = "репозиторий изменён во время обновления"
		return res
	}
	if err := commitHelmIndex(e.Name, tmp); err != nil {
		res.Error = err.Error()
		return res
	}
	res.Charts = countHelmCharts(idx)
	return res
}

// ─── каталог ───

func (m *Manager) SearchHelmCharts(query, repoName string) (*HelmCatalogResult, error) {
	if repoName != "" {
		if err := validateHelmRepoName(repoName); err != nil {
			return nil, err
		}
	}
	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err != nil {
		return nil, err
	}
	terms := strings.Fields(strings.ToLower(query))
	var all []HelmCatalogChart
	for _, e := range f.Repositories {
		if e == nil || validateHelmRepoName(e.Name) != nil || (repoName != "" && e.Name != repoName) {
			continue
		}
		idx, _, err := loadHelmIndex(e.Name)
		if err != nil {
			continue // индекс не скачан или повреждён: виден в списке репозиториев
		}
		for name, versions := range idx.Entries {
			if len(versions) == 0 || versions[0] == nil || versions[0].Metadata == nil {
				continue
			}
			latest := versions[0] // IndexFile сортирует версии по убыванию
			if !helmMatches(terms, name, latest.Description, latest.Keywords) {
				continue
			}
			all = append(all, HelmCatalogChart{
				Repo: e.Name, Name: name, Version: latest.Version, AppVersion: latest.AppVersion,
				Description: latest.Description, Deprecated: latest.Deprecated, Versions: len(versions),
			})
		}
	}
	sort.Slice(all, func(i, j int) bool {
		if all[i].Name != all[j].Name {
			return all[i].Name < all[j].Name
		}
		return all[i].Repo < all[j].Repo
	})
	res := &HelmCatalogResult{Total: len(all), Items: all}
	if len(all) > helmSearchMaxResults {
		res.Items = all[:helmSearchMaxResults]
	}
	if res.Items == nil {
		res.Items = []HelmCatalogChart{}
	}
	return res, nil
}

func helmMatches(terms []string, name, desc string, keywords []string) bool {
	if len(terms) == 0 {
		return true
	}
	hay := strings.ToLower(name + " " + desc + " " + strings.Join(keywords, " "))
	for _, t := range terms {
		if !strings.Contains(hay, t) {
			return false
		}
	}
	return true
}

func helmRepoEntry(name string) (*repo.Entry, error) {
	if err := validateHelmRepoName(name); err != nil {
		return nil, err
	}
	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err != nil {
		return nil, err
	}
	e := f.Get(name)
	if e == nil {
		return nil, fmt.Errorf("репозиторий %q не найден", name)
	}
	return e, nil
}

func (m *Manager) GetHelmChartVersions(repoName, chartName string) ([]HelmChartVersionInfo, error) {
	if _, err := helmRepoEntry(repoName); err != nil {
		return nil, err
	}
	idx, _, err := loadHelmIndex(repoName)
	if err != nil {
		return nil, fmt.Errorf("индекс репозитория %q недоступен: %w", repoName, err)
	}
	versions := idx.Entries[chartName]
	if len(versions) == 0 {
		return nil, fmt.Errorf("чарт %q не найден в репозитории %q", chartName, repoName)
	}
	out := make([]HelmChartVersionInfo, 0, len(versions))
	for _, v := range versions {
		if v == nil || v.Metadata == nil {
			continue
		}
		created := ""
		if !v.Created.IsZero() {
			created = v.Created.UTC().Format(time.RFC3339)
		}
		out = append(out, HelmChartVersionInfo{
			Version: v.Version, AppVersion: v.AppVersion, Created: created,
			Description: v.Description, Deprecated: v.Deprecated,
		})
	}
	return out, nil
}

// resolveChartURL превращает ссылку из index.yaml (может быть относительной) в абсолютную http(s).
func resolveChartURL(repoURL, ref string) (string, error) {
	base, err := url.Parse(repoURL + "/")
	if err != nil {
		return "", err
	}
	r, err := url.Parse(ref)
	if err != nil {
		return "", fmt.Errorf("некорректная ссылка на чарт: %w", err)
	}
	abs := base.ResolveReference(r)
	if abs.Scheme != "http" && abs.Scheme != "https" {
		return "", fmt.Errorf("ссылка на чарт со схемой %q не поддерживается", abs.Scheme)
	}
	abs.User = nil
	return abs.String(), nil
}

// lookupHelmChartVersion находит запись чарта в индексе репозитория (version == "" — последняя).
func lookupHelmChartVersion(repoName, chartName, version string) (*repo.Entry, *repo.ChartVersion, error) {
	e, err := helmRepoEntry(repoName)
	if err != nil {
		return nil, nil, err
	}
	idx, _, err := loadHelmIndex(repoName)
	if err != nil {
		return nil, nil, fmt.Errorf("индекс репозитория %q недоступен: %w", repoName, err)
	}
	cv, err := idx.Get(chartName, version)
	if err != nil {
		return nil, nil, err
	}
	if len(cv.URLs) == 0 {
		return nil, nil, fmt.Errorf("у чарта %s %s нет ссылки на архив", chartName, cv.Version)
	}
	return e, cv, nil
}

// downloadHelmChart скачивает архив чарта (с лимитом размера), сверяет digest из index.yaml и разбирает чарт.
func downloadHelmChart(ctx context.Context, e *repo.Entry, cv *repo.ChartVersion) (*chartv2.Chart, error) {
	chartURL, err := resolveChartURL(e.URL, cv.URLs[0])
	if err != nil {
		return nil, err
	}
	data, err := helmGet(ctx, e, chartURL, helmChartMaxBytes)
	if err != nil {
		return nil, err
	}
	if cv.Digest != "" {
		sum := sha256.Sum256(data)
		if !strings.EqualFold(hex.EncodeToString(sum[:]), cv.Digest) {
			return nil, errors.New("контрольная сумма архива не совпадает с index.yaml")
		}
	}
	c, err := loader.LoadArchive(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("не удалось разобрать архив чарта: %w", err)
	}
	return c, nil
}

// GetHelmChartPackage скачивает архив чарта из репозитория и отдаёт README и values.yaml.
func (m *Manager) GetHelmChartPackage(repoName, chartName, version string) (*HelmChartPackage, error) {
	e, cv, err := lookupHelmChartVersion(repoName, chartName, version)
	if err != nil {
		return nil, err
	}

	key := repoName + "/" + chartName + "/" + cv.Version + "/" + cv.Digest
	helmPkgMu.Lock()
	cached := helmPkgCache[key]
	helmPkgMu.Unlock()
	if cached != nil {
		return cached, nil
	}

	c, err := downloadHelmChart(context.Background(), e, cv)
	if err != nil {
		return nil, err
	}

	pkg := &HelmChartPackage{
		Repo: repoName, Name: chartName, Version: cv.Version, AppVersion: cv.AppVersion,
		Description: cv.Description, Home: cv.Home, Sources: cv.Sources, Keywords: cv.Keywords,
		Deprecated: cv.Deprecated,
	}
	for _, mt := range cv.Maintainers {
		if mt == nil {
			continue
		}
		s := mt.Name
		if mt.Email != "" {
			s += " <" + mt.Email + ">"
		}
		pkg.Maintainers = append(pkg.Maintainers, s)
	}
	for _, f := range c.Raw {
		switch {
		case f.Name == "values.yaml":
			pkg.Values = clipText(f.Data)
		case pkg.Readme == "" && strings.EqualFold(f.Name, "README.md"):
			pkg.Readme = clipText(f.Data)
		}
	}

	helmPkgMu.Lock()
	if _, ok := helmPkgCache[key]; !ok {
		helmPkgCache[key] = pkg
		helmPkgOrder = append(helmPkgOrder, key)
		if len(helmPkgOrder) > helmPackageCacheSize {
			delete(helmPkgCache, helmPkgOrder[0])
			helmPkgOrder = helmPkgOrder[1:]
		}
	}
	helmPkgMu.Unlock()
	return pkg, nil
}

func clipText(b []byte) string {
	if len(b) > helmPackageTextMax {
		return string(b[:helmPackageTextMax]) + "\n… (обрезано)"
	}
	return string(b)
}
