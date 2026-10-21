package k8s

import (
	"context"
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
)

func newArgoDyn(objs ...runtime.Object) *dynamicfake.FakeDynamicClient {
	return dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{kindTable["application"].gvr: "ApplicationList"}, objs...)
}

func TestApplySyncOperation(t *testing.T) {
	t.Run("занято", func(t *testing.T) {
		obj := argoApp("argocd", "a", map[string]interface{}{"operation": map[string]interface{}{"sync": map[string]interface{}{}}})
		if err := applySyncOperation(obj, ArgoSyncOptions{}); err == nil {
			t.Fatal("ожидалась ошибка: операция уже идёт")
		}
	})
	t.Run("идёт фаза Running", func(t *testing.T) {
		obj := argoApp("argocd", "a", map[string]interface{}{"status": map[string]interface{}{
			"operationState": map[string]interface{}{"phase": "Running"}}})
		if err := applySyncOperation(obj, ArgoSyncOptions{}); err == nil {
			t.Fatal("ожидалась ошибка")
		}
	})
	t.Run("параметры попадают в операцию", func(t *testing.T) {
		obj := argoApp("argocd", "a", map[string]interface{}{"spec": map[string]interface{}{
			"syncPolicy": map[string]interface{}{"syncOptions": []interface{}{"CreateNamespace=true"}}}})
		if err := applySyncOperation(obj, ArgoSyncOptions{Revision: "main", Prune: true, Force: true, DryRun: true}); err != nil {
			t.Fatal(err)
		}
		if !boolAt(obj.Object, "operation", "sync", "prune") || !boolAt(obj.Object, "operation", "sync", "dryRun") {
			t.Errorf("prune/dryRun: %v", obj.Object["operation"])
		}
		if strAt(obj.Object, "operation", "sync", "revision") != "main" {
			t.Error("revision")
		}
		if !boolAt(obj.Object, "operation", "sync", "syncStrategy", "hook", "force") {
			t.Error("force")
		}
		if so, _ := sliceAt(obj.Object, "operation", "sync", "syncOptions"); len(so) != 1 {
			t.Errorf("syncOptions: %v", so)
		}
		if strAt(obj.Object, "operation", "initiatedBy", "username") != argoInitiator {
			t.Error("initiatedBy")
		}
	})
	t.Run("revision у multi-source", func(t *testing.T) {
		obj := argoApp("argocd", "a", map[string]interface{}{"spec": map[string]interface{}{
			"sources": []interface{}{map[string]interface{}{"repoURL": "x"}}}})
		if err := applySyncOperation(obj, ArgoSyncOptions{Revision: "v1"}); err == nil {
			t.Fatal("ожидался отказ")
		}
		if err := applySyncOperation(obj, ArgoSyncOptions{}); err != nil {
			t.Fatalf("без revision должно работать: %v", err)
		}
	})
}

func TestApplyTerminate(t *testing.T) {
	running := argoApp("argocd", "a", map[string]interface{}{"status": map[string]interface{}{
		"operationState": map[string]interface{}{"phase": "Running"}}})
	if err := applyTerminate(running); err != nil {
		t.Fatal(err)
	}
	if got := strAt(running.Object, "status", "operationState", "phase"); got != "Terminating" {
		t.Errorf("phase = %q", got)
	}

	pending := argoApp("argocd", "b", map[string]interface{}{"operation": map[string]interface{}{"sync": map[string]interface{}{}}})
	if err := applyTerminate(pending); err != nil {
		t.Fatal(err)
	}
	if _, found, _ := unstructured.NestedFieldNoCopy(pending.Object, "operation"); found {
		t.Error("operation должна быть удалена")
	}

	idle := argoApp("argocd", "c", map[string]interface{}{"status": map[string]interface{}{
		"operationState": map[string]interface{}{"phase": "Succeeded"}}})
	if err := applyTerminate(idle); err == nil {
		t.Error("нечего останавливать: ожидалась ошибка")
	}
}

func TestApplyRollback(t *testing.T) {
	history := []interface{}{
		map[string]interface{}{"id": int64(3), "revision": "r3",
			"source": map[string]interface{}{"repoURL": "https://h/r", "targetRevision": "main"}},
	}
	status := map[string]interface{}{"history": history}

	obj := argoApp("argocd", "a", map[string]interface{}{"status": status})
	if err := applyRollback(obj, 3, true, false); err != nil {
		t.Fatal(err)
	}
	if strAt(obj.Object, "operation", "sync", "revision") != "r3" ||
		strAt(obj.Object, "operation", "sync", "source", "repoURL") != "https://h/r" ||
		!boolAt(obj.Object, "operation", "sync", "prune") {
		t.Errorf("operation: %v", obj.Object["operation"])
	}

	if err := applyRollback(argoApp("argocd", "b", map[string]interface{}{"status": status}), 99, false, false); err == nil {
		t.Error("неизвестный id: ожидалась ошибка")
	}

	auto := argoApp("argocd", "c", map[string]interface{}{
		"status": status,
		"spec":   map[string]interface{}{"syncPolicy": map[string]interface{}{"automated": map[string]interface{}{}}},
	})
	if err := applyRollback(auto, 3, false, false); err == nil {
		t.Error("при auto-sync откат должен быть запрещён")
	}
}

func TestSyncAndRefreshArgoApps(t *testing.T) {
	ctx := context.Background()
	dyn := newArgoDyn(argoApp("argocd", "a", nil))
	gvr := kindTable["application"].gvr

	if _, err := syncArgoApps(ctx, dyn, []ArgoAppRef{{"argocd", "a"}, {"argocd", "b"}}, ArgoSyncOptions{Revision: "x"}); err == nil {
		t.Error("revision для нескольких приложений должен отклоняться")
	}

	res, err := syncArgoApps(ctx, dyn, []ArgoAppRef{{"argocd", "a"}, {"argocd", "missing"}, {"argocd", "a"}}, ArgoSyncOptions{Prune: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(res) != 2 || res[0].Error != "" || res[1].Error == "" {
		t.Fatalf("результаты (дубликат схлопнут, отсутствующее — ошибка): %+v", res)
	}
	got, err := dyn.Resource(gvr).Namespace("argocd").Get(ctx, "a", metav1.GetOptions{})
	if err != nil {
		t.Fatal(err)
	}
	if !boolAt(got.Object, "operation", "sync", "prune") {
		t.Errorf("операция не записана: %v", got.Object)
	}
	// повторный sync, пока операция не снята, отклоняется
	if res, _ := syncArgoApps(ctx, dyn, []ArgoAppRef{{"argocd", "a"}}, ArgoSyncOptions{}); res[0].Error == "" {
		t.Error("ожидался отказ при уже запрошенной операции")
	}

	if _, err := refreshArgoApps(ctx, dyn, []ArgoAppRef{{"argocd", "a"}}, true); err != nil {
		t.Fatal(err)
	}
	got, _ = dyn.Resource(gvr).Namespace("argocd").Get(ctx, "a", metav1.GetOptions{})
	if got.GetAnnotations()[argoRefreshAnnotation] != "hard" {
		t.Errorf("аннотации: %v", got.GetAnnotations())
	}
}

func TestNormalizeArgoRefs(t *testing.T) {
	if _, err := normalizeArgoRefs(nil); err == nil {
		t.Error("пустой список")
	}
	if _, err := normalizeArgoRefs([]ArgoAppRef{{"", "x"}}); err == nil {
		t.Error("пустой namespace")
	}
	got, err := normalizeArgoRefs([]ArgoAppRef{{" a ", "x"}, {"a", "x"}, {"a", "y"}})
	if err != nil || len(got) != 2 {
		t.Errorf("%v, %v", got, err)
	}
}
