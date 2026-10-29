// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"helm.sh/helm/v4/pkg/action"
	ci "helm.sh/helm/v4/pkg/chart"
	chartv2 "helm.sh/helm/v4/pkg/chart/v2"
	"helm.sh/helm/v4/pkg/kube"
	helmrelease "helm.sh/helm/v4/pkg/release"
	"sigs.k8s.io/yaml"
)

const (
	// helmValuesMaxBytes — предел размера values, присланных из UI.
	helmValuesMaxBytes = 1 << 20
	helmUpgradeTimeout = 5 * time.Minute
)

// HelmUpgradePreview — результат dry-run апгрейда values (чарт остаётся прежним;
// смена версии чарта — HelmChartUpgradePreview).
type HelmUpgradePreview struct {
	CurrentValues   string `json:"currentValues"`   // user-supplied values текущей ревизии
	NewValues       string `json:"newValues"`       // нормализованные новые values
	CurrentManifest string `json:"currentManifest"` // значения Secret скрыты
	NewManifest     string `json:"newManifest"`     // значения Secret скрыты
}

// parseHelmValues разбирает YAML values из UI. Пустой ввод = пустой map.
func parseHelmValues(src string) (map[string]interface{}, error) {
	if len(src) > helmValuesMaxBytes {
		return nil, fmt.Errorf("values are larger than %d KiB", helmValuesMaxBytes>>10)
	}
	if strings.TrimSpace(src) == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal([]byte(src), &out); err != nil {
		return nil, fmt.Errorf("invalid YAML: %w", err)
	}
	if out == nil {
		out = map[string]interface{}{}
	}
	return out, nil
}

func marshalHelmValues(v map[string]interface{}) (string, error) {
	if len(v) == 0 {
		return "", nil
	}
	b, err := yaml.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}

// helmUpgrade применяет (или в dry-run только считает) апгрейд релиза с новыми user-values.
// target == nil — чарт берётся ИЗ САМОГО РЕЛИЗА (апгрейд только values); иначе релиз переводится
// на переданный чарт (смена версии, см. helm_chart_upgrade.go).
// ResetValues=true: новые values заменяют прежние целиком, поэтому ключ, удалённый
// в редакторе, возвращается к дефолту чарта (при смене версии — к дефолту НОВОГО чарта).
func (m *Manager) helmUpgrade(ctx context.Context, namespace, name string, target *chartv2.Chart, vals map[string]interface{}, dryRun bool, logHandler slog.Handler) (cur, next helmrelease.Accessor, curVals map[string]interface{}, err error) {
	cfg, err := m.helmConfigForRelease(ctx, namespace, name, logHandler)
	if err != nil {
		return nil, nil, nil, err
	}
	curRel, err := action.NewGet(cfg).Run(name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("release %s/%s not found: %w", namespace, name, err)
	}
	if cur, err = helmrelease.NewAccessor(curRel); err != nil {
		return nil, nil, nil, err
	}
	if curVals, err = action.NewGetValues(cfg).Run(name); err != nil {
		return nil, nil, nil, err
	}

	up := action.NewUpgrade(cfg)
	up.Namespace = namespace
	up.ResetValues = true
	up.Timeout = helmUpgradeTimeout
	if dryRun {
		up.DryRunStrategy = action.DryRunClient
	} else {
		up.WaitStrategy = kube.StatusWatcherStrategy
	}

	var chrt ci.Charter = cur.Chart()
	if target != nil {
		chrt = target
	}
	res, err := up.RunWithContext(ctx, name, chrt, vals)
	if err != nil {
		return nil, nil, nil, err
	}
	if next, err = helmrelease.NewAccessor(res); err != nil {
		return nil, nil, nil, err
	}
	return cur, next, curVals, nil
}

// PreviewHelmUpgrade — dry-run: в кластер и в историю ничего не пишется.
func (m *Manager) PreviewHelmUpgrade(namespace, name, valuesYAML string) (*HelmUpgradePreview, error) {
	if err := validateHelmRef(namespace, name); err != nil {
		return nil, err
	}
	vals, err := parseHelmValues(valuesYAML)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	cur, next, curVals, err := m.helmUpgrade(ctx, namespace, name, nil, vals, true, nil)
	if err != nil {
		return nil, fmt.Errorf("preview %s/%s: %w", namespace, name, err)
	}
	curY, err := marshalHelmValues(curVals)
	if err != nil {
		return nil, err
	}
	newY, err := marshalHelmValues(vals)
	if err != nil {
		return nil, err
	}
	return &HelmUpgradePreview{
		CurrentValues:   curY,
		NewValues:       newY,
		CurrentManifest: redactHelmManifest(cur.Manifest()),
		NewManifest:     redactHelmManifest(next.Manifest()),
	}, nil
}

// UpgradeHelmRelease применяет новые values и возвращает номер созданной ревизии.
// opID — идентификатор операции для событий helm:progress и отмены (пустой — без отслеживания).
func (m *Manager) UpgradeHelmRelease(namespace, name, valuesYAML, opID string) (rev int, err error) {
	if err := validateHelmRef(namespace, name); err != nil {
		return 0, err
	}
	vals, err := parseHelmValues(valuesYAML)
	if err != nil {
		return 0, err
	}
	ctx, tr, err := m.beginHelmOp(context.Background(), opID, "upgrade", namespace, name, true)
	if err != nil {
		return 0, err
	}
	defer func() { tr.finish(err) }()
	ctx, cancel := context.WithTimeout(ctx, helmUpgradeTimeout+30*time.Second)
	defer cancel()

	tr.note("values change")
	tr.setPhase("Applying changes and waiting for readiness…")
	_, next, _, err := m.helmUpgrade(ctx, namespace, name, nil, vals, false, tr.handler())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("upgrade of %s/%s did not finish within %s", namespace, name, helmUpgradeTimeout)
		}
		return 0, fmt.Errorf("upgrade %s/%s: %w", namespace, name, err)
	}
	tr.setRevision(next.Version())
	return next.Version(), nil
}
