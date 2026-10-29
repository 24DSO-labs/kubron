// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"sync/atomic"
	"testing"
	"time"

	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

func cvs(versions ...string) repo.ChartVersions {
	out := make(repo.ChartVersions, 0, len(versions))
	for _, v := range versions {
		out = append(out, &repo.ChartVersion{Metadata: &chartv2.Metadata{Name: "web", Version: v}})
	}
	return out // порядок как в индексе Helm: по убыванию версии
}

func idxWith(name string, versions ...string) helmRepoIndex {
	return helmRepoIndex{name: name, idx: &repo.IndexFile{Entries: map[string]repo.ChartVersions{"web": cvs(versions...)}}}
}

func TestFindHelmUpdate(t *testing.T) {
	r := idxWith("bitnami", "2.0.0", "1.5.0", "1.4.0", "1.3.0-rc.1")

	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "1.4.0"); u == nil || u.Latest != "2.0.0" || u.Repo != "bitnami" {
		t.Errorf("update to 2.0.0: %+v", u)
	}
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "2.0.0"); u != nil {
		t.Errorf("already the latest version: %+v", u)
	}
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "v1.5.0"); u == nil || u.Latest != "2.0.0" {
		t.Errorf("version with a v prefix: %+v", u)
	}
	// установленной версии нет в индексе: источник неизвестен, ничего не предлагаем
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "0.9.0"); u != nil {
		t.Errorf("unknown source: %+v", u)
	}
	if u := findHelmUpdate([]helmRepoIndex{r}, "other", "1.0.0"); u != nil {
		t.Errorf("chart is not in the repository: %+v", u)
	}
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "not-a-version"); u != nil {
		t.Errorf("invalid version: %+v", u)
	}
}

func TestFindHelmUpdatePrerelease(t *testing.T) {
	r := idxWith("repo", "3.0.0-beta.1", "2.0.0", "1.0.0")
	// стабильная версия не получает пререлиз
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "1.0.0"); u == nil || u.Latest != "2.0.0" {
		t.Errorf("stable version: %+v", u)
	}
	// пререлиз видит более новые пререлизы
	if u := findHelmUpdate([]helmRepoIndex{r}, "web", "2.0.0"); u != nil {
		t.Errorf("a 3.0.0-beta.1 prerelease must not be offered on 2.0.0: %+v", u)
	}
	rp := idxWith("repo", "3.0.0-beta.2", "3.0.0-beta.1", "2.0.0")
	if u := findHelmUpdate([]helmRepoIndex{rp}, "web", "3.0.0-beta.1"); u == nil || u.Latest != "3.0.0-beta.2" {
		t.Errorf("prerelease to prerelease: %+v", u)
	}
}

// Источник — тот репозиторий, где есть установленная версия, а не любой с чартом того же имени.
func TestFindHelmUpdateUsesOriginRepo(t *testing.T) {
	a := idxWith("a", "9.0.0", "1.0.0")
	b := idxWith("b", "1.2.0", "1.1.0")
	u := findHelmUpdate([]helmRepoIndex{a, b}, "web", "1.1.0")
	if u == nil || u.Repo != "b" || u.Latest != "1.2.0" {
		t.Errorf("origin=b: %+v", u)
	}
}

func TestHelmCacheDebounce(t *testing.T) {
	var n atomic.Int32
	c := &helmCache{debounce: 30 * time.Millisecond, onChange: func() { n.Add(1) }}
	for i := 0; i < 20; i++ {
		c.changed()
	}
	time.Sleep(120 * time.Millisecond)
	if got := n.Load(); got != 1 {
		t.Fatalf("a burst of changes must coalesce into one notification, got %d", got)
	}
	c.changed()
	time.Sleep(120 * time.Millisecond)
	if got := n.Load(); got != 2 {
		t.Fatalf("new change after a pause: %d", got)
	}
}

func TestHelmCacheStopCancelsPending(t *testing.T) {
	var n atomic.Int32
	c := &helmCache{debounce: 40 * time.Millisecond, onChange: func() { n.Add(1) }}
	c.changed()
	c.stop()
	c.changed() // после stop не должно срабатывать
	time.Sleep(120 * time.Millisecond)
	if got := n.Load(); got != 0 {
		t.Fatalf("there must be no notifications after stop, got %d", got)
	}
}
