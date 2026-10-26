// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import "testing"

func sample(v float64, kv ...string) promSample {
	l := map[string]string{}
	for i := 0; i+1 < len(kv); i += 2 {
		l[kv[i]] = kv[i+1]
	}
	return promSample{Labels: l, Value: v}
}

func TestBuildHotPods(t *testing.T) {
	ratio := []promSample{
		sample(0.5, "namespace", "a", "pod", "low"),
		sample(0.95, "namespace", "a", "pod", "hot"),
		sample(0.8, "namespace", "b", "pod", "mid"),
		sample(0.7, "namespace", "b", "pod", "x"),
		sample(0.9, "namespace", "b"), // без имени пода пропускается
	}
	usage := []promSample{sample(950, "namespace", "a", "pod", "hot"), sample(800, "namespace", "b", "pod", "mid")}
	got := buildHotPods(ratio, usage)
	if len(got) != hotspotsTop || got[0].Name != "hot" || got[0].Percent != 95 || got[0].Used != 950 || got[1].Name != "mid" {
		t.Errorf("%+v", got)
	}
}

func TestBuildHotVolumes(t *testing.T) {
	got := buildHotVolumes(
		[]promSample{sample(0.4, "namespace", "a", "persistentvolumeclaim", "data"), sample(0.9, "namespace", "a", "persistentvolumeclaim", "wal")},
		[]promSample{sample(100, "namespace", "a", "persistentvolumeclaim", "wal")},
	)
	if len(got) != 2 || got[0].Claim != "wal" || got[0].Used != 90 || got[1].Capacity != 0 {
		t.Errorf("%+v", got)
	}
}
