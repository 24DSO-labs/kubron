// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/json"
	"reflect"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func helmSecretFor(t *testing.T, p map[string]any) corev1.Secret {
	t.Helper()
	raw, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	_, _ = gz.Write(raw)
	_ = gz.Close()
	return corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{Name: "sh.helm.release.v1.web.v1", Namespace: "prod"},
		Type:       helmSecretType,
		Data:       map[string][]byte{"release": []byte(base64.StdEncoding.EncodeToString(buf.Bytes()))},
	}
}

const testHelmManifest = `---
# Source: app/templates/dep.yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: web
---
# Source: app/templates/sec.yaml
apiVersion: v1
kind: Secret
metadata:
  name: db
data:
  password: UzNDUkVU
`

func releasePayload(version int, status string, extra map[string]any) map[string]any {
	p := map[string]any{
		"name": "web", "namespace": "prod", "version": version,
		"info":  map[string]any{"status": status, "description": "d"},
		"chart": map[string]any{"metadata": map[string]any{"name": "app", "version": "1.0.0"}},
	}
	for k, v := range extra {
		p[k] = v
	}
	return p
}

func TestBuildHelmDetail(t *testing.T) {
	secs := []corev1.Secret{
		helmSecretFor(t, releasePayload(1, "superseded", nil)),
		helmSecretFor(t, releasePayload(2, "deployed", map[string]any{"manifest": testHelmManifest})),
	}
	d, err := buildHelmDetail(secs, "prod", "web")
	if err != nil {
		t.Fatal(err)
	}
	if len(d.History) != 2 || d.History[0].Revision != 2 || d.History[1].Status != "superseded" {
		t.Errorf("history: %+v", d.History)
	}
	if len(d.Resources) != 2 || d.Resources[0].Kind != "Deployment" || d.Resources[0].Namespace != "prod" || d.Resources[1].Kind != "Secret" {
		t.Errorf("resources: %+v", d.Resources)
	}
	if _, err := buildHelmDetail(nil, "prod", "web"); err == nil {
		t.Error("без ревизий ожидалась ошибка")
	}
}

func TestMergeHelmValues(t *testing.T) {
	chart := map[string]any{"a": map[string]any{"b": 1, "c": 2}, "x": 1}
	cfg := map[string]any{"a": map[string]any{"b": 5}, "x": nil, "y": "new"}
	got := mergeHelmValues(chart, cfg)
	want := map[string]any{"a": map[string]any{"b": 5, "c": 2}, "y": "new"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}
	if chart["a"].(map[string]any)["b"] != 1 || chart["x"] != 1 {
		t.Error("исходные значения чарта не должны меняться")
	}
}

func TestRedactHelmManifest(t *testing.T) {
	out := redactHelmManifest(testHelmManifest)
	if strings.Contains(out, "UzNDUkVU") || !strings.Contains(out, argoSecretRedacted) {
		t.Errorf("значение секрета не скрыто:\n%s", out)
	}
	if !strings.Contains(out, "kind: Deployment") || !strings.Contains(out, "# Source: app/templates/sec.yaml") {
		t.Errorf("остальное должно сохраниться:\n%s", out)
	}
}

func TestHelmContentFrom(t *testing.T) {
	sec := helmSecretFor(t, releasePayload(3, "deployed", map[string]any{
		"config":   map[string]any{"replicas": 3},
		"manifest": testHelmManifest,
		"info":     map[string]any{"status": "deployed", "notes": "hello"},
		"chart": map[string]any{
			"metadata": map[string]any{"name": "app", "version": "1.0.0"},
			"values":   map[string]any{"image": "nginx", "replicas": 1},
		},
	}))
	c, err := helmContentFrom(&sec)
	if err != nil {
		t.Fatal(err)
	}
	if c.Revision != 3 || c.Notes != "hello" || !strings.Contains(c.Values, "replicas: 3") || strings.Contains(c.Values, "image") {
		t.Errorf("values: %+v", c)
	}
	if !strings.Contains(c.ComputedValues, "image: nginx") || !strings.Contains(c.ComputedValues, "replicas: 3") {
		t.Errorf("computed: %q", c.ComputedValues)
	}
}

func TestValidateHelmRef(t *testing.T) {
	for _, bad := range [][2]string{{"", "web"}, {"prod", ""}, {"prod", "a,owner=x"}, {"Prod", "web"}} {
		if validateHelmRef(bad[0], bad[1]) == nil {
			t.Errorf("%v должен отклоняться", bad)
		}
	}
	if err := validateHelmRef("prod", "my-app.v2"); err != nil {
		t.Error(err)
	}
}
