// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"strings"

	"github.com/Masterminds/semver/v3"
	repo "helm.sh/helm/v4/pkg/repo/v1"
)

// helmUpdate — доступное обновление чарта установленного релиза.
type helmUpdate struct {
	Latest     string
	Repo       string
	Deprecated bool
}

type helmRepoIndex struct {
	name string
	idx  *repo.IndexFile
}

// helmAllIndexes снимает индексы всех добавленных репозиториев (HTTP и OCI) один раз.
func helmAllIndexes() []helmRepoIndex {
	var repos []helmRepoIndex
	helmRepoMu.Lock()
	f, err := loadHelmRepoFile()
	helmRepoMu.Unlock()
	if err == nil {
		for _, e := range f.Repositories {
			if e == nil || validateHelmRepoName(e.Name) != nil {
				continue
			}
			if idx, _, err := loadHelmIndex(e.Name); err == nil {
				repos = append(repos, helmRepoIndex{name: e.Name, idx: idx})
			}
		}
	}
	return append(repos, ociIndexes()...)
}

// helmUpdateFinder снимает индексы добавленных репозиториев один раз и возвращает функцию поиска
// обновления для (чарт, установленная версия).
//
// ИСТОЧНИК РЕЛИЗА определяется по наличию установленной версии в индексе репозитория: Helm не хранит,
// из какого репозитория поставлен релиз, а чарты с одинаковым именем в разных репозиториях — разные чарты.
// Если установленной версии нет ни в одном индексе (старая версия вычищена или чарт ставили вручную),
// обновление НЕ показывается: лучше промолчать, чем предложить чужой чарт.
// Пререлизы предлагаются только тем, кто сам стоит на пререлизе.
func helmUpdateFinder() func(chart, installed string) *helmUpdate {
	repos := helmAllIndexes()
	return func(chart, installed string) *helmUpdate {
		return findHelmUpdate(repos, chart, installed)
	}
}

func findHelmUpdate(repos []helmRepoIndex, chart, installed string) *helmUpdate {
	cur, err := semver.NewVersion(installed)
	if err != nil {
		return nil
	}
	for _, r := range repos {
		versions := r.idx.Entries[chart]
		if !helmHasVersion(versions, cur) {
			continue
		}
		// версии отсортированы по убыванию: берём первую подходящую
		for _, cv := range versions {
			if cv == nil || cv.Metadata == nil {
				continue
			}
			v, err := semver.NewVersion(cv.Version)
			if err != nil {
				continue
			}
			if v.Prerelease() != "" && cur.Prerelease() == "" {
				continue
			}
			if v.GreaterThan(cur) {
				return &helmUpdate{Latest: strings.TrimPrefix(cv.Version, "v"), Repo: r.name, Deprecated: cv.Deprecated}
			}
			break // дальше только более старые версии
		}
		return nil // релиз «принадлежит» этому репозиторию, обновлений нет
	}
	return nil
}

func helmHasVersion(versions repo.ChartVersions, cur *semver.Version) bool {
	for _, cv := range versions {
		if cv == nil || cv.Metadata == nil {
			continue
		}
		if v, err := semver.NewVersion(cv.Version); err == nil && v.Equal(cur) {
			return true
		}
	}
	return false
}
