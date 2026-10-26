// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestRolloutChanges(t *testing.T) {
	now := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	yes := true
	mk := func(name, ownerKind string, age time.Duration) appsv1.ReplicaSet {
		return appsv1.ReplicaSet{ObjectMeta: metav1.ObjectMeta{
			Name: name, Namespace: "ns",
			CreationTimestamp: metav1.NewTime(now.Add(-age)),
			Annotations:       map[string]string{"deployment.kubernetes.io/revision": "7"},
			OwnerReferences:   []metav1.OwnerReference{{Kind: ownerKind, Name: "web", Controller: &yes}},
		}, Spec: appsv1.ReplicaSetSpec{Template: corev1.PodTemplateSpec{Spec: corev1.PodSpec{Containers: []corev1.Container{{Image: "web:1.2"}}}}}}
	}
	got := rolloutChanges([]appsv1.ReplicaSet{
		mk("new", "Deployment", 10*time.Minute),
		mk("old", "Deployment", 3*time.Hour),
		mk("other", "StatefulSet", time.Minute),
	}, now.Add(-time.Hour))
	if len(got) != 1 || got[0].Name != "web" || got[0].Detail != "revision 7 · web:1.2" || got[0].Target != "deployment" {
		t.Errorf("%+v", got)
	}
}

func TestFinishChanges(t *testing.T) {
	base := time.Date(2026, 10, 23, 12, 0, 0, 0, time.UTC)
	var in []ChangeEvent
	for i := 0; i < maxChanges+3; i++ {
		in = append(in, ChangeEvent{Name: "x", at: base.Add(time.Duration(i) * time.Minute)})
	}
	got := finishChanges(in)
	if len(got) != maxChanges || !got[0].at.After(got[1].at) || got[0].Time == "" {
		t.Errorf("len=%d first=%+v", len(got), got[0])
	}
	if finishChanges(nil) == nil {
		t.Error("пустой результат должен быть [], а не null")
	}
}
