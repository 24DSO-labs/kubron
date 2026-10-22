package k8s

import (
	"encoding/json"
	"fmt"
	"reflect"
	"strings"
	"testing"
)

func mustJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func deploy(replicas int) map[string]any {
	return map[string]any{
		"apiVersion": "apps/v1", "kind": "Deployment",
		"metadata": map[string]any{"name": "web", "namespace": "prod"},
		"spec":     map[string]any{"replicas": replicas},
	}
}

func TestUnifiedDiffFormat(t *testing.T) {
	got := unifiedDiff([]string{"a", "b", "c"}, []string{"a", "x", "c"}, 3)
	want := "@@ -1,3 +1,3 @@\n a\n-b\n+x\n c"
	if got != want {
		t.Errorf("got:\n%s\nwant:\n%s", got, want)
	}
	if d := unifiedDiff([]string{"a"}, []string{"a"}, 3); d != "" {
		t.Errorf("одинаковые входы дают пустой diff, получили %q", d)
	}
	// Два далёких изменения при ctx=1 — два hunk'а, строки между ними не попадают.
	a := []string{"1", "2", "3", "4", "5", "6", "7", "8", "9"}
	b := []string{"X", "2", "3", "4", "5", "6", "7", "8", "Y"}
	d := unifiedDiff(a, b, 1)
	if strings.Count(d, "@@") != 4 || strings.Contains(d, " 5") {
		t.Errorf("ожидалось 2 hunk'а без средней части:\n%s", d)
	}
}

func TestLineDiffReconstructsBothSides(t *testing.T) {
	cases := [][2][]string{
		{{}, {"a", "b"}},
		{{"a", "b"}, {}},
		{{"a", "b", "c", "d"}, {"b", "d", "e"}},
		{{"x", "x", "x"}, {"x"}},
		{{"a", "b", "a", "b"}, {"b", "a", "b", "a"}},
	}
	for i, c := range cases {
		var left, right []string
		for _, op := range lineDiff(c[0], c[1]) {
			if op.kind != '+' {
				left = append(left, op.line)
			}
			if op.kind != '-' {
				right = append(right, op.line)
			}
		}
		if !reflect.DeepEqual(append([]string{}, left...), append([]string{}, c[0]...)) && !(len(left) == 0 && len(c[0]) == 0) {
			t.Errorf("случай %d: левая часть %v != %v", i, left, c[0])
		}
		if !reflect.DeepEqual(append([]string{}, right...), append([]string{}, c[1]...)) && !(len(right) == 0 && len(c[1]) == 0) {
			t.Errorf("случай %d: правая часть %v != %v", i, right, c[1])
		}
	}
}

func TestBuildArgoDiffStates(t *testing.T) {
	items := []argoManagedResource{
		{Group: "apps", Kind: "Deployment", Namespace: "prod", Name: "same",
			NormalizedLiveState: mustJSON(t, deploy(2)), PredictedLiveState: mustJSON(t, deploy(2))},
		{Group: "apps", Kind: "Deployment", Namespace: "prod", Name: "mod", Hook: true,
			NormalizedLiveState: mustJSON(t, deploy(2)), PredictedLiveState: mustJSON(t, deploy(3))},
		{Kind: "ConfigMap", Namespace: "prod", Name: "new", LiveState: "null", TargetState: mustJSON(t, map[string]any{"kind": "ConfigMap"})},
		{Kind: "ConfigMap", Namespace: "prod", Name: "old", LiveState: mustJSON(t, map[string]any{"kind": "ConfigMap"}), TargetState: ""},
		{Kind: "ConfigMap", Name: "ghost", LiveState: "null", TargetState: "null"}, // нет ни там, ни там: пропускается
	}
	d := buildArgoDiff(items)

	if d.Total != 4 || d.Changed != 3 || d.Truncated {
		t.Fatalf("total=%d changed=%d truncated=%v", d.Total, d.Changed, d.Truncated)
	}
	states := map[string]string{}
	for _, r := range d.Resources {
		states[r.Name] = r.State
	}
	want := map[string]string{"same": ArgoDiffInSync, "mod": ArgoDiffModified, "new": ArgoDiffMissing, "old": ArgoDiffExtra}
	if !reflect.DeepEqual(states, want) {
		t.Errorf("states = %v, want %v", states, want)
	}
	// Различающиеся ресурсы идут первыми, InSync — в конце.
	if d.Resources[len(d.Resources)-1].State != ArgoDiffInSync {
		t.Errorf("InSync должен быть последним: %+v", d.Resources)
	}
	for _, r := range d.Resources {
		if r.Name == "mod" {
			if !r.Hook || !strings.Contains(r.Diff, "-  replicas: 2") || !strings.Contains(r.Diff, "+  replicas: 3") {
				t.Errorf("diff Modified: hook=%v\n%s", r.Hook, r.Diff)
			}
		}
		if r.State == ArgoDiffInSync && r.Diff != "" {
			t.Errorf("у InSync diff должен быть пуст: %q", r.Diff)
		}
	}
}

func TestDiffResourceFallsBackToLiveAndTargetState(t *testing.T) {
	r, ok := diffResource(argoManagedResource{
		Kind: "Deployment", Name: "w", LiveState: mustJSON(t, deploy(1)), TargetState: mustJSON(t, deploy(5)),
	})
	if !ok || r.State != ArgoDiffModified {
		t.Fatalf("без normalized/predicted нужно сравнивать live и target: %+v", r)
	}
}

func TestDiffIgnoresManagedFieldsAndStatusNoise(t *testing.T) {
	live := deploy(2)
	live["status"] = map[string]any{"readyReplicas": 2}
	live["metadata"].(map[string]any)["managedFields"] = []any{map[string]any{"manager": "kubectl"}}

	r, ok := diffResource(argoManagedResource{Kind: "Deployment", Name: "w", NormalizedLiveState: mustJSON(t, live), PredictedLiveState: mustJSON(t, deploy(2))})
	if !ok || r.State != ArgoDiffInSync {
		t.Errorf("managedFields и status не должны давать различия: %+v", r)
	}
}

func TestDiffRedactsSecretValues(t *testing.T) {
	secret := func(data map[string]any, ann string) map[string]any {
		m := map[string]any{
			"apiVersion": "v1", "kind": "Secret",
			"metadata":   map[string]any{"name": "db", "annotations": map[string]any{argoLastAppliedAnnKey: ann, "keep": "me"}},
			"data":       data,
			"stringData": map[string]any{"plain": "S3CRET-PLAIN-" + fmt.Sprint(len(data))},
		}
		return m
	}
	markers := []string{"S3CRET-OLD", "S3CRET-NEW", "S3CRET-SAME", "S3CRET-PLAIN", "S3CRET-ANN", "S3CRET-ONLYLIVE", "S3CRET-ONLYDESIRED"}
	liveM := secret(map[string]any{"pass": "S3CRET-OLD", "same": "S3CRET-SAME", "gone": "S3CRET-ONLYLIVE"}, `{"data":{"pass":"S3CRET-ANN"}}`)
	desM := secret(map[string]any{"pass": "S3CRET-NEW", "same": "S3CRET-SAME", "added": "S3CRET-ONLYDESIRED"}, `{"data":{"pass":"S3CRET-ANN"}}`)

	r, ok := diffResource(argoManagedResource{
		Kind: "Secret", Namespace: "prod", Name: "db",
		NormalizedLiveState: mustJSON(t, liveM), PredictedLiveState: mustJSON(t, desM),
	})
	if !ok || r.State != ArgoDiffModified {
		t.Fatalf("изменение значения секрета должно быть видно как Modified: %+v", r)
	}
	for _, m := range markers {
		if strings.Contains(r.Diff, m) {
			t.Errorf("значение %q секрета попало в diff:\n%s", m, r.Diff)
		}
	}
	// Различие видно, значения нет; ключи сохранены.
	for _, want := range []string{"REDACTED (current)", "REDACTED (desired)", "gone:", "added:"} {
		if !strings.Contains(r.Diff, want) {
			t.Errorf("в diff нет %q:\n%s", want, r.Diff)
		}
	}
	if strings.Contains(r.Diff, "last-applied") {
		t.Errorf("last-applied-configuration (копия данных) должен быть убран:\n%s", r.Diff)
	}

	// Одинаковые значения: InSync, и ничего не утекает.
	same := secret(map[string]any{"pass": "S3CRET-SAME"}, `{"x":"S3CRET-ANN"}`)
	rs, _ := diffResource(argoManagedResource{Kind: "Secret", Name: "db", NormalizedLiveState: mustJSON(t, same), PredictedLiveState: mustJSON(t, same)})
	if rs.State != ArgoDiffInSync || rs.Diff != "" {
		t.Errorf("одинаковый секрет: %+v", rs)
	}

	// Секрет, которого нет на одной из сторон (Missing/Extra), тоже редактируется.
	only, _ := diffResource(argoManagedResource{Kind: "Secret", Name: "db", LiveState: "null", TargetState: mustJSON(t, liveM)})
	if only.State != ArgoDiffMissing || strings.Contains(only.Diff, "S3CRET") {
		t.Errorf("Missing-секрет: state=%s\n%s", only.State, only.Diff)
	}
	extra, _ := diffResource(argoManagedResource{Kind: "Secret", Name: "db", LiveState: mustJSON(t, desM)})
	if extra.State != ArgoDiffExtra || strings.Contains(extra.Diff, "S3CRET") {
		t.Errorf("Extra-секрет: state=%s\n%s", extra.State, extra.Diff)
	}

	// Весь результат сериализуется без секретов.
	raw, _ := json.Marshal(buildArgoDiff([]argoManagedResource{{Kind: "Secret", Name: "db", NormalizedLiveState: mustJSON(t, liveM), PredictedLiveState: mustJSON(t, desM)}}))
	for _, m := range markers {
		if strings.Contains(string(raw), m) {
			t.Errorf("значение %q в JSON-ответе: %s", m, raw)
		}
	}
}

func TestDiffNonSecretKindsAreNotRedacted(t *testing.T) {
	cm := func(v string) string {
		return mustJSON(t, map[string]any{"kind": "ConfigMap", "data": map[string]any{"k": v}})
	}
	r, _ := diffResource(argoManagedResource{Kind: "ConfigMap", Name: "c", NormalizedLiveState: cm("old"), PredictedLiveState: cm("new")})
	if !strings.Contains(r.Diff, "old") || !strings.Contains(r.Diff, "new") {
		t.Errorf("ConfigMap не секрет, значения должны быть видны:\n%s", r.Diff)
	}
	// Kind Secret в другой группе (CRD) — не core Secret: правило касается только core.
	crd, _ := diffResource(argoManagedResource{Group: "example.com", Kind: "Secret", Name: "c", NormalizedLiveState: cm("old"), PredictedLiveState: cm("new")})
	if !strings.Contains(crd.Diff, "old") {
		t.Errorf("Secret из чужой группы не должен редактироваться:\n%s", crd.Diff)
	}
}

func TestDiffHugeManifestIsTruncatedNotComputed(t *testing.T) {
	data := map[string]any{}
	for i := 0; i < argoDiffMaxLines+10; i++ {
		data[fmt.Sprintf("key-%05d", i)] = "v"
	}
	big := func(extra string) string {
		d := map[string]any{"extra": extra}
		for k, v := range data {
			d[k] = v
		}
		return mustJSON(t, map[string]any{"kind": "ConfigMap", "data": d})
	}
	r, _ := diffResource(argoManagedResource{Kind: "ConfigMap", Name: "big", NormalizedLiveState: big("a"), PredictedLiveState: big("b")})
	if r.State != ArgoDiffModified || !r.Truncated || !strings.Contains(r.Diff, "слишком большой") {
		t.Errorf("state=%s truncated=%v diff=%q", r.State, r.Truncated, r.Diff)
	}
}

func TestBuildArgoDiffResourceLimit(t *testing.T) {
	items := make([]argoManagedResource, argoDiffMaxResources+5)
	for i := range items {
		items[i] = argoManagedResource{Kind: "ConfigMap", Name: fmt.Sprintf("c-%04d", i), LiveState: "null", TargetState: mustJSON(t, map[string]any{"kind": "ConfigMap"})}
	}
	d := buildArgoDiff(items)
	if !d.Truncated || len(d.Resources) != argoDiffMaxResources || d.Total != len(items) {
		t.Errorf("truncated=%v shown=%d total=%d", d.Truncated, len(d.Resources), d.Total)
	}
}
