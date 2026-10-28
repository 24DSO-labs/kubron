// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"time"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/client-go/kubernetes"
)

// Хранилища релизов Helm (H7). Helm v3/v4 умеет хранить ревизии в Secret (по умолчанию) и в ConfigMap
// (HELM_DRIVER=configmap). Формат одинаков: объект `sh.helm.release.v1.<имя>.v<ревизия>` с меткой owner=helm
// и полем data.release = base64(gzip(JSON)). Kubron читает оба хранилища и помнит, где лежит каждый релиз:
// операции (откат, удаление, апгрейд) должны идти через тот же драйвер, иначе Helm не найдёт релиз.
const (
	helmStorageSecret    = "secret"
	helmStorageConfigMap = "configmap"
)

// helmStored — ревизия релиза в сыром виде, не зависящая от типа объекта-носителя.
type helmStored struct {
	Name      string // имя объекта (sh.helm.release.v1.<имя>.v<N>)
	Namespace string
	Storage   string // helmStorageSecret | helmStorageConfigMap
	Raw       []byte // содержимое поля release (base64 от gzip-JSON); nil — поля нет
}

// helmStoredFromSecret: только настоящие секреты релизов (тип helm.sh/release.v1).
func helmStoredFromSecret(sec *corev1.Secret) (helmStored, bool) {
	if string(sec.Type) != helmSecretType {
		return helmStored{}, false
	}
	return helmStored{Name: sec.Name, Namespace: sec.Namespace, Storage: helmStorageSecret, Raw: sec.Data["release"]}, true
}

// helmStoredFromConfigMap: у ConfigMap нет типа, поэтому признаком служит поле release
// (метку owner=helm отбирает запрос).
func helmStoredFromConfigMap(cm *corev1.ConfigMap) (helmStored, bool) {
	raw, ok := cm.Data["release"]
	if !ok {
		return helmStored{}, false
	}
	return helmStored{Name: cm.Name, Namespace: cm.Namespace, Storage: helmStorageConfigMap, Raw: []byte(raw)}, true
}

// decodeHelmRaw распаковывает release (base64 → gzip → JSON) в out.
func decodeHelmRaw(raw []byte, ident string, out any) error {
	if raw == nil {
		return fmt.Errorf("%s: нет поля release", ident)
	}
	decoded, err := base64.StdEncoding.DecodeString(string(raw))
	if err != nil {
		return fmt.Errorf("%s: %w", ident, err)
	}
	gz, err := gzip.NewReader(bytes.NewReader(decoded))
	if err != nil {
		return fmt.Errorf("%s: %w", ident, err)
	}
	defer gz.Close()

	// Защита от «gzip-бомбы» в подменённом объекте
	jsonBytes, err := io.ReadAll(io.LimitReader(gz, 64<<20))
	if err != nil {
		return err
	}
	return json.Unmarshal(jsonBytes, out)
}

func (s helmStored) decode(out any) error {
	return decodeHelmRaw(s.Raw, s.Storage+" "+s.Namespace+"/"+s.Name, out)
}

// listHelmStored собирает ревизии из Secret и ConfigMap (секреты идут первыми).
// Отказ в доступе к одному из хранилищ терпим, если второе прочиталось: у ограниченного пользователя
// может не быть права на configmaps (или на secrets) во всём кластере. Любая другая ошибка — ошибка.
func listHelmStored(ctx context.Context, c kubernetes.Interface, namespace, selector string) ([]helmStored, error) {
	var out []helmStored
	opts := metav1.ListOptions{LabelSelector: selector}

	var errS, errC error
	sl, errS := c.CoreV1().Secrets(namespace).List(ctx, opts)
	if errS == nil {
		for i := range sl.Items {
			if s, ok := helmStoredFromSecret(&sl.Items[i]); ok {
				out = append(out, s)
			}
		}
	}
	cl, errC := c.CoreV1().ConfigMaps(namespace).List(ctx, opts)
	if errC == nil {
		for i := range cl.Items {
			if s, ok := helmStoredFromConfigMap(&cl.Items[i]); ok {
				out = append(out, s)
			}
		}
	}

	switch {
	case errS != nil && errC != nil:
		return nil, errS
	case errS != nil && !apierrors.IsForbidden(errS):
		return nil, errS
	case errC != nil && !apierrors.IsForbidden(errC):
		return nil, errC
	}
	return out, nil
}

func helmReleaseSelector(name string) string {
	set := labels.Set{"owner": "helm"}
	if name != "" {
		set["name"] = name
	}
	return labels.SelectorFromSet(set).String()
}

// getHelmStored читает конкретную ревизию: сначала Secret, затем ConfigMap.
func getHelmStored(ctx context.Context, c kubernetes.Interface, namespace, name string, revision int) (helmStored, error) {
	key := helmSecretName(name, revision)
	sec, err := c.CoreV1().Secrets(namespace).Get(ctx, key, metav1.GetOptions{})
	if err == nil {
		if s, ok := helmStoredFromSecret(sec); ok {
			return s, nil
		}
		return helmStored{}, fmt.Errorf("секрет %s не является релизом Helm", key)
	}
	if !apierrors.IsNotFound(err) && !apierrors.IsForbidden(err) {
		return helmStored{}, err
	}
	cm, errC := c.CoreV1().ConfigMaps(namespace).Get(ctx, key, metav1.GetOptions{})
	if errC == nil {
		if s, ok := helmStoredFromConfigMap(cm); ok {
			return s, nil
		}
		return helmStored{}, fmt.Errorf("configmap %s не является релизом Helm", key)
	}
	if apierrors.IsNotFound(errC) && apierrors.IsNotFound(err) {
		return helmStored{}, apierrors.NewNotFound(corev1.Resource("secrets"), key)
	}
	return helmStored{}, err
}

// latestHelmPayloads — последняя ревизия каждого релиза (superseded не входят).
// Если один и тот же релиз найден в обоих хранилищах (миграция), побеждает большая ревизия,
// при равенстве — Secret (он идёт в списке первым).
func latestHelmPayloads(items []helmStored) []helmPayload {
	latest := make(map[string]helmPayload)
	for _, it := range items {
		var p helmPayload
		if err := it.decode(&p); err != nil {
			continue // битая ревизия не должна ломать весь список
		}
		p.Storage = it.Storage
		key := p.Namespace + "/" + p.Name
		if prev, ok := latest[key]; !ok || p.Version > prev.Version {
			latest[key] = p
		}
	}
	out := make([]helmPayload, 0, len(latest))
	for _, p := range latest {
		if isSuperseded(p.Info.Status) {
			continue
		}
		out = append(out, p)
	}
	return out
}

// pickHelmStorage определяет, где лежит последняя ревизия релиза. Нет релиза — пустая строка.
func pickHelmStorage(items []helmStored) string {
	best, storage := -1, ""
	for _, it := range items {
		var p struct {
			Version int `json:"version"`
		}
		if it.decode(&p) != nil {
			continue
		}
		if p.Version > best {
			best, storage = p.Version, it.Storage
		}
	}
	return storage
}

// helmStorageOf — где лежит релиз ("" — не найден). Нужен операциям, чтобы выбрать драйвер Helm.
func (m *Manager) helmStorageOf(ctx context.Context, namespace, name string) (string, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return "", err
	}
	ctx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	items, err := listHelmStored(ctx, conn.Client, namespace, helmReleaseSelector(name))
	if err != nil {
		return "", err
	}
	return pickHelmStorage(items), nil
}

// helmDriverName — значение для action.Configuration.Init.
func helmDriverName(storage string) (string, error) {
	switch storage {
	case "", helmStorageSecret:
		return "secret", nil
	case helmStorageConfigMap:
		return "configmap", nil
	}
	return "", fmt.Errorf("неизвестное хранилище релизов %q (допустимы secret и configmap)", storage)
}
