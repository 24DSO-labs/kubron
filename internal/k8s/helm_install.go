// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/action"
	ci "helm.sh/helm/v4/pkg/chart"
	chartutil "helm.sh/helm/v4/pkg/chart/v2/util"
	"helm.sh/helm/v4/pkg/kube"
	helmrelease "helm.sh/helm/v4/pkg/release"
	"k8s.io/apimachinery/pkg/util/validation"
)

const helmInstallTimeout = 5 * time.Minute

// HelmInstallRequest — параметры установки чарта из репозитория каталога.
type HelmInstallRequest struct {
	Repo            string `json:"repo"`
	Chart           string `json:"chart"`
	Version         string `json:"version"`
	ReleaseName     string `json:"releaseName"`
	Namespace       string `json:"namespace"`
	Values          string `json:"values"` // YAML с переопределениями (не полные values чарта)
	CreateNamespace bool   `json:"createNamespace"`
	Wait            bool   `json:"wait"`    // ждать готовности ресурсов (до 5 минут)
	Storage         string `json:"storage"` // secret (по умолчанию) | configmap — где Helm хранит ревизии релиза
	OpID            string `json:"opId"`    // идентификатор операции для helm:progress и отмены (необязателен)
}

// HelmInstallResult — итог установки или dry-run.
type HelmInstallResult struct {
	ReleaseName string `json:"releaseName"`
	Namespace   string `json:"namespace"`
	Revision    int    `json:"revision"`
	Status      string `json:"status"`
	Manifest    string `json:"manifest"` // Secret-ы замаскированы
	Notes       string `json:"notes"`
}

// validateHelmInstallRequest проверяет вход до любых сетевых вызовов.
func validateHelmInstallRequest(r *HelmInstallRequest) error {
	if err := validateHelmRepoName(r.Repo); err != nil {
		return err
	}
	if r.Chart == "" || len(r.Chart) > 253 || strings.ContainsAny(r.Chart, "\r\n") {
		return errors.New("не указан чарт")
	}
	if err := chartutil.ValidateReleaseName(r.ReleaseName); err != nil {
		return fmt.Errorf("имя релиза %q: %w", r.ReleaseName, err)
	}
	if r.Namespace == "" || len(validation.IsDNS1123Label(r.Namespace)) > 0 {
		return fmt.Errorf("некорректный namespace %q", r.Namespace)
	}
	if _, err := helmDriverName(r.Storage); err != nil {
		return err
	}
	return nil
}

// helmInstall скачивает чарт из репозитория и ставит его (или в dry-run только рендерит).
// dry-run использует DryRunClient: кластер не опрашивается и ничего не создаётся, поэтому
// lookup-функции шаблонов вернут пустоту, а проверки существующих ресурсов не выполняются.
func (m *Manager) helmInstall(ctx context.Context, req HelmInstallRequest, dryRun bool) (res *HelmInstallResult, err error) {
	if err := validateHelmInstallRequest(&req); err != nil {
		return nil, err
	}
	vals, err := parseHelmValues(req.Values)
	if err != nil {
		return nil, err
	}

	var tr *helmTracker
	if !dryRun { // dry-run короткий и в прогрессе не нуждается
		ctx, tr, err = m.beginHelmOp(ctx, req.OpID, "install", req.Namespace, req.ReleaseName, true)
		if err != nil {
			return nil, err
		}
		defer func() { tr.finish(err) }()
	}

	tr.note(fmt.Sprintf("%s/%s %s", req.Repo, req.Chart, req.Version))
	tr.setPhase("Загрузка чарта…")
	chrt, err := loadHelmChart(ctx, req.Repo, req.Chart, req.Version)
	if err != nil {
		return nil, err
	}

	ac, err := ci.NewAccessor(chrt)
	if err != nil {
		return nil, err
	}
	if ac.IsLibraryChart() {
		return nil, errors.New("library-чарт нельзя установить как релиз")
	}
	if deps := ac.MetaDependencies(); len(deps) > 0 {
		if err := action.CheckDependencies(chrt, deps); err != nil {
			return nil, fmt.Errorf("в архиве чарта нет зависимостей: %w", err)
		}
	}

	if !dryRun {
		// Helm проверяет занятость имени только в выбранном хранилище: без этой проверки релиз с тем же
		// именем мог бы появиться одновременно в Secret и в ConfigMap.
		if st, err := m.helmStorageOf(ctx, req.Namespace, req.ReleaseName); err != nil {
			return nil, fmt.Errorf("не удалось проверить имя релиза: %w", err)
		} else if st != "" {
			return nil, fmt.Errorf("релиз %s/%s уже существует (хранилище: %s)", req.Namespace, req.ReleaseName, st)
		}
	}
	cfg, err := m.helmConfigDriver(req.Namespace, tr.handler(), req.Storage)
	if err != nil {
		return nil, err
	}
	in := action.NewInstall(cfg)
	in.ReleaseName = req.ReleaseName
	in.Namespace = req.Namespace
	in.Timeout = helmInstallTimeout
	if dryRun {
		in.DryRunStrategy = action.DryRunClient
	} else {
		in.CreateNamespace = req.CreateNamespace
		if req.Wait {
			in.WaitStrategy = kube.StatusWatcherStrategy
		} else {
			in.WaitStrategy = kube.HookOnlyStrategy
		}
	}

	ctx, cancel := context.WithTimeout(ctx, helmInstallTimeout+30*time.Second)
	defer cancel()

	tr.setPhase("Установка и ожидание готовности…")
	rel, err := in.RunWithContext(ctx, chrt, vals)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return nil, fmt.Errorf("установка %s/%s не завершилась за %s", req.Namespace, req.ReleaseName, helmInstallTimeout)
		}
		return nil, fmt.Errorf("установка %s/%s: %w", req.Namespace, req.ReleaseName, err)
	}
	acc, err := helmrelease.NewAccessor(rel)
	if err != nil {
		return nil, err
	}
	tr.setRevision(acc.Version())
	return &HelmInstallResult{
		ReleaseName: acc.Name(),
		Namespace:   acc.Namespace(),
		Revision:    acc.Version(),
		Status:      acc.Status(),
		Manifest:    redactHelmManifest(acc.Manifest()),
		Notes:       acc.Notes(),
	}, nil
}

// PreviewHelmInstall — dry-run установки: манифест и notes, без изменений в кластере.
func (m *Manager) PreviewHelmInstall(req HelmInstallRequest) (*HelmInstallResult, error) {
	return m.helmInstall(context.Background(), req, true)
}

// InstallHelmChart устанавливает чарт; блокируется до завершения (при Wait — до 5 минут).
func (m *Manager) InstallHelmChart(req HelmInstallRequest) (*HelmInstallResult, error) {
	return m.helmInstall(context.Background(), req, false)
}
