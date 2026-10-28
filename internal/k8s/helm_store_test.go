// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes/fake"
	clienttesting "k8s.io/client-go/testing"
)

var helmOwnerLabels = map[string]string{"owner": "helm", "name": "web"}

// helmSecretObj — секрет релиза web указанной ревизии с меткой owner=helm.
func helmSecretObj(t *testing.T, version int, status string, extra map[string]any) *corev1.Secret {
	t.Helper()
	sec := helmSecretFor(t, releasePayload(version, status, extra))
	sec.Name = helmSecretName("web", version)
	sec.Labels = helmOwnerLabels
	return &sec
}

// helmConfigMapObj — тот же релиз, но в ConfigMap (так хранит Helm при HELM_DRIVER=configmap).
func helmConfigMapObj(t *testing.T, version int, status string, extra map[string]any) *corev1.ConfigMap {
	t.Helper()
	sec := helmSecretFor(t, releasePayload(version, status, extra))
	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: helmSecretName("web", version), Namespace: "prod", Labels: helmOwnerLabels},
		Data:       map[string]string{"release": string(sec.Data["release"])},
	}
}

func forbidden(resource string) clienttesting.ReactionFunc {
	return func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: resource}, "", nil)
	}
}

func TestListHelmStoredReadsBothStorages(t *testing.T) {
	notRelease := &corev1.ConfigMap{ // метка есть, поля release нет — не релиз
		ObjectMeta: metav1.ObjectMeta{Name: "other", Namespace: "prod", Labels: map[string]string{"owner": "helm"}},
		Data:       map[string]string{"x": "y"},
	}
	wrongType := helmSecretObj(t, 9, "deployed", nil) // секрет с меткой, но другого типа
	wrongType.Name, wrongType.Type = "sh.helm.release.v1.web.v9", corev1.SecretTypeOpaque

	c := fake.NewSimpleClientset(helmSecretObj(t, 1, "superseded", nil), helmConfigMapObj(t, 2, "deployed", nil), notRelease, wrongType)
	items, err := listHelmStored(context.Background(), c, "", helmReleaseSelector(""))
	if err != nil {
		t.Fatal(err)
	}
	if len(items) != 2 || items[0].Storage != helmStorageSecret || items[1].Storage != helmStorageConfigMap {
		t.Fatalf("items: %+v", items)
	}
	// фильтр по имени релиза
	items, err = listHelmStored(context.Background(), c, "prod", helmReleaseSelector("nope"))
	if err != nil || len(items) != 0 {
		t.Errorf("чужое имя: %+v %v", items, err)
	}
}

func TestListHelmStoredToleratesForbiddenOnOneSide(t *testing.T) {
	ctx := context.Background()

	cs := fake.NewSimpleClientset(helmSecretObj(t, 1, "deployed", nil), helmConfigMapObj(t, 2, "deployed", nil))
	cs.PrependReactor("list", "configmaps", forbidden("configmaps"))
	items, err := listHelmStored(ctx, cs, "", helmReleaseSelector(""))
	if err != nil || len(items) != 1 || items[0].Storage != helmStorageSecret {
		t.Errorf("нет доступа к configmaps: %+v %v", items, err)
	}

	cs = fake.NewSimpleClientset(helmSecretObj(t, 1, "deployed", nil), helmConfigMapObj(t, 2, "deployed", nil))
	cs.PrependReactor("list", "secrets", forbidden("secrets"))
	items, err = listHelmStored(ctx, cs, "", helmReleaseSelector(""))
	if err != nil || len(items) != 1 || items[0].Storage != helmStorageConfigMap {
		t.Errorf("нет доступа к secrets: %+v %v", items, err)
	}

	cs = fake.NewSimpleClientset()
	cs.PrependReactor("list", "secrets", forbidden("secrets"))
	cs.PrependReactor("list", "configmaps", forbidden("configmaps"))
	if _, err := listHelmStored(ctx, cs, "", helmReleaseSelector("")); err == nil {
		t.Error("нет доступа ни к чему — должна быть ошибка")
	}

	// не-Forbidden ошибка нельзя глотать: список был бы неполным
	cs = fake.NewSimpleClientset(helmSecretObj(t, 1, "deployed", nil))
	cs.PrependReactor("list", "configmaps", func(clienttesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewInternalError(context.DeadlineExceeded)
	})
	if _, err := listHelmStored(ctx, cs, "", helmReleaseSelector("")); err == nil {
		t.Error("внутренняя ошибка API должна возвращаться")
	}
}

func TestLatestHelmPayloads(t *testing.T) {
	// ревизия 1 в Secret (superseded), ревизия 2 в ConfigMap — актуальна именно она
	c := fake.NewSimpleClientset(helmSecretObj(t, 1, "superseded", nil), helmConfigMapObj(t, 2, "deployed", nil))
	items, _ := listHelmStored(context.Background(), c, "", helmReleaseSelector(""))
	got := latestHelmPayloads(items)
	if len(got) != 1 || got[0].Version != 2 || got[0].Storage != helmStorageConfigMap {
		t.Fatalf("got %+v", got)
	}

	// одна и та же ревизия в обоих хранилищах: побеждает Secret
	c = fake.NewSimpleClientset(helmSecretObj(t, 3, "deployed", nil), helmConfigMapObj(t, 3, "deployed", nil))
	items, _ = listHelmStored(context.Background(), c, "", helmReleaseSelector(""))
	if got := latestHelmPayloads(items); len(got) != 1 || got[0].Storage != helmStorageSecret {
		t.Errorf("ничья: %+v", got)
	}

	// единственная ревизия superseded не показывается; битый объект пропускается
	bad := helmStored{Name: "x", Namespace: "prod", Storage: helmStorageConfigMap, Raw: []byte("not-base64!")}
	one := []helmStored{bad}
	if got := latestHelmPayloads(one); len(got) != 0 {
		t.Errorf("битый объект: %+v", got)
	}
}

func TestPickHelmStorage(t *testing.T) {
	c := fake.NewSimpleClientset(helmSecretObj(t, 1, "superseded", nil), helmConfigMapObj(t, 2, "deployed", nil))
	items, _ := listHelmStored(context.Background(), c, "prod", helmReleaseSelector("web"))
	if got := pickHelmStorage(items); got != helmStorageConfigMap {
		t.Errorf("последняя ревизия в configmap, получено %q", got)
	}
	if got := pickHelmStorage(nil); got != "" {
		t.Errorf("релиза нет: %q", got)
	}
}

func TestGetHelmStoredFallsBackToConfigMap(t *testing.T) {
	ctx := context.Background()
	c := fake.NewSimpleClientset(helmConfigMapObj(t, 2, "deployed", nil))
	it, err := getHelmStored(ctx, c, "prod", "web", 2)
	if err != nil || it.Storage != helmStorageConfigMap {
		t.Fatalf("%+v %v", it, err)
	}
	if _, err := getHelmStored(ctx, c, "prod", "web", 7); !apierrors.IsNotFound(err) {
		t.Errorf("ожидалась NotFound: %v", err)
	}
	// секрет приоритетнее
	c = fake.NewSimpleClientset(helmSecretObj(t, 1, "deployed", nil), helmConfigMapObj(t, 1, "deployed", nil))
	if it, err := getHelmStored(ctx, c, "prod", "web", 1); err != nil || it.Storage != helmStorageSecret {
		t.Errorf("приоритет secret: %+v %v", it, err)
	}
}

func TestHelmDetailAndContentAcrossStorages(t *testing.T) {
	c := fake.NewSimpleClientset(
		helmSecretObj(t, 1, "superseded", nil),
		helmConfigMapObj(t, 2, "deployed", map[string]any{
			"manifest": testHelmManifest,
			"config":   map[string]any{"replicas": 5},
			"info":     map[string]any{"status": "deployed", "notes": "hi"},
		}),
	)
	items, err := listHelmStored(context.Background(), c, "prod", helmReleaseSelector("web"))
	if err != nil {
		t.Fatal(err)
	}
	d, err := buildHelmDetailStored(items, "prod", "web")
	if err != nil || len(d.History) != 2 || d.History[0].Revision != 2 || len(d.Resources) != 2 {
		t.Fatalf("detail: %+v %v", d, err)
	}
	content, err := helmContentFromStored(items[1])
	if err != nil || content.Revision != 2 || content.Notes != "hi" || !strings.Contains(content.Values, "replicas: 5") {
		t.Fatalf("content: %+v %v", content, err)
	}
	// в манифесте значения Secret скрываются и для ConfigMap-хранилища
	if strings.Contains(content.Manifest, "UzNDUkVU") {
		t.Error("значение Secret не должно попадать в манифест")
	}
}

func TestHelmDriverName(t *testing.T) {
	for in, want := range map[string]string{"": "secret", "secret": "secret", "configmap": "configmap"} {
		if got, err := helmDriverName(in); err != nil || got != want {
			t.Errorf("%q: %q %v", in, got, err)
		}
	}
	for _, bad := range []string{"sql", "memory", "ConfigMap", "secrets "} {
		if _, err := helmDriverName(bad); err == nil {
			t.Errorf("%q должно отклоняться", bad)
		}
	}
}

func TestInstallRequestValidatesStorage(t *testing.T) {
	r := validInstallReq()
	r.Storage = "configmap"
	if err := validateHelmInstallRequest(&r); err != nil {
		t.Errorf("configmap: %v", err)
	}
	r.Storage = "sql"
	if err := validateHelmInstallRequest(&r); err == nil {
		t.Error("неизвестное хранилище должно отклоняться")
	}
}
