// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestConvertPodStatusTerminating(t *testing.T) {
	pod := corev1.Pod{Status: corev1.PodStatus{Phase: corev1.PodRunning}}
	if got := convertPod(pod).Status; got != "Running" {
		t.Fatalf("regular pod: %q", got)
	}
	now := metav1.NewTime(time.Now())
	pod.DeletionTimestamp = &now
	if got := convertPod(pod).Status; got != "Terminating" {
		t.Errorf("pod being deleted: %q, expected Terminating", got)
	}
}
