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
	"helm.sh/helm/v4/pkg/kube"
	helmrelease "helm.sh/helm/v4/pkg/release"
	"sigs.k8s.io/yaml"
)

const (
	// helmValuesMaxBytes — предел размера values, присланных из UI.
	helmValuesMaxBytes = 1 << 20
	helmUpgradeTimeout = 5 * time.Minute
)

// HelmUpgradePreview — результат dry-run апгрейда values (чарт остаётся прежним).
type HelmUpgradePreview struct {
	CurrentValues   string `json:"currentValues"`   // user-supplied values текущей ревизии
	NewValues       string `json:"newValues"`       // нормализованные новые values
	CurrentManifest string `json:"currentManifest"` // значения Secret скрыты
	NewManifest     string `json:"newManifest"`     // значения Secret скрыты
}

// parseHelmValues разбирает YAML values из UI. Пустой ввод = пустой map.
func parseHelmValues(src string) (map[string]interface{}, error) {
	if len(src) > helmValuesMaxBytes {
		return nil, fmt.Errorf("values больше %d КиБ", helmValuesMaxBytes>>10)
	}
	if strings.TrimSpace(src) == "" {
		return map[string]interface{}{}, nil
	}
	var out map[string]interface{}
	if err := yaml.Unmarshal([]byte(src), &out); err != nil {
		return nil, fmt.Errorf("некорректный YAML: %w", err)
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

// helmUpgradeValues применяет (или в dry-run только считает) апгрейд релиза с новыми
// user-values. Чарт берётся ИЗ САМОГО РЕЛИЗА: смена версии чарта — этап H4/H5.
// ResetValues=true: новые values заменяют прежние целиком, поэтому ключ, удалённый
// в редакторе, возвращается к дефолту чарта.
func (m *Manager) helmUpgradeValues(ctx context.Context, namespace, name string, vals map[string]interface{}, dryRun bool, logHandler slog.Handler) (cur, next helmrelease.Accessor, curVals map[string]interface{}, err error) {
	cfg, err := m.helmConfigForRelease(ctx, namespace, name, logHandler)
	if err != nil {
		return nil, nil, nil, err
	}
	curRel, err := action.NewGet(cfg).Run(name)
	if err != nil {
		return nil, nil, nil, fmt.Errorf("релиз %s/%s не найден: %w", namespace, name, err)
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

	res, err := up.RunWithContext(ctx, name, cur.Chart(), vals)
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

	cur, next, curVals, err := m.helmUpgradeValues(ctx, namespace, name, vals, true, nil)
	if err != nil {
		return nil, fmt.Errorf("предпросмотр %s/%s: %w", namespace, name, err)
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

	tr.note("изменение values")
	tr.setPhase("Применение изменений и ожидание готовности…")
	_, next, _, err := m.helmUpgradeValues(ctx, namespace, name, vals, false, tr.handler())
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			return 0, fmt.Errorf("апгрейд %s/%s не завершился за %s", namespace, name, helmUpgradeTimeout)
		}
		return 0, fmt.Errorf("апгрейд %s/%s: %w", namespace, name, err)
	}
	tr.setRevision(next.Version())
	return next.Version(), nil
}
