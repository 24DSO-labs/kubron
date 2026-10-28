// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var t0 = time.Date(2026, 10, 21, 12, 0, 0, 0, time.UTC)

func ago(d time.Duration) metav1.Time { return metav1.NewTime(t0.Add(-d)) }

func pod(name string, phase corev1.PodPhase, age time.Duration) corev1.Pod {
	return corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: "ns", CreationTimestamp: ago(age)},
		Status:     corev1.PodStatus{Phase: phase},
	}
}

func TestAnalyzePods(t *testing.T) {
	crash := pod("crash", corev1.PodRunning, time.Hour)
	crash.Status.ContainerStatuses = []corev1.ContainerStatus{{
		RestartCount: 9,
		State:        corev1.ContainerState{Waiting: &corev1.ContainerStateWaiting{Reason: "CrashLoopBackOff"}},
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			Reason: "OOMKilled", FinishedAt: ago(10 * time.Minute)}},
	}}
	oldRestart := pod("old", corev1.PodRunning, 72*time.Hour)
	oldRestart.Status.ContainerStatuses = []corev1.ContainerStatus{{
		LastTerminationState: corev1.ContainerState{Terminated: &corev1.ContainerStateTerminated{
			Reason: "Error", FinishedAt: ago(30 * time.Hour)}},
	}}
	freshPending := pod("fresh", corev1.PodPending, time.Minute)
	stuckPending := pod("stuck", corev1.PodPending, time.Hour)
	stuckPending.Status.Conditions = []corev1.PodCondition{{
		Type: corev1.PodScheduled, Status: corev1.ConditionFalse, Reason: corev1.PodReasonUnschedulable, LastTransitionTime: ago(time.Hour)}}
	notReady := pod("notready", corev1.PodRunning, time.Hour)
	notReady.Status.Conditions = []corev1.PodCondition{{Type: corev1.PodReady, Status: corev1.ConditionFalse, LastTransitionTime: ago(10 * time.Minute)}}
	done := pod("done", corev1.PodSucceeded, time.Hour)
	evictedOld := pod("evicted-old", corev1.PodFailed, 48*time.Hour)
	terminating := crash
	terminating.Name = "term"
	terminating.DeletionTimestamp = &metav1.Time{Time: t0}

	got := analyzePods([]corev1.Pod{crash, oldRestart, freshPending, stuckPending, notReady, done, evictedOld, terminating}, t0)

	if got.Restarts != 1 || got.OOMKilled != 1 {
		t.Errorf("restarts=%d oom=%d, want 1/1 (restart 30h ago is outside the window)", got.Restarts, got.OOMKilled)
	}
	var names []string
	for _, p := range got.Problems {
		names = append(names, p.Name+":"+p.Reason)
	}
	want := []string{"crash:CrashLoopBackOff", "stuck:Unschedulable", "notready:NotReady"}
	if len(names) != len(want) || got.ProblemsTotal != len(want) {
		t.Fatalf("problems=%v total=%d, want %v", names, got.ProblemsTotal, want)
	}
	for i := range want {
		if names[i] != want[i] {
			t.Errorf("problems[%d]=%s, want %s", i, names[i], want[i])
		}
	}
	if got.Problems[0].Restarts != 9 {
		t.Errorf("pod restarts = %d, want 9", got.Problems[0].Restarts)
	}
}

func TestAnalyzePodsLimitsList(t *testing.T) {
	var pods []corev1.Pod
	for i := 0; i < maxProblemPods+5; i++ {
		p := pod(string(rune('a'+i)), corev1.PodPending, time.Hour)
		pods = append(pods, p)
	}
	got := analyzePods(pods, t0)
	if len(got.Problems) != maxProblemPods || got.ProblemsTotal != maxProblemPods+5 {
		t.Errorf("shown=%d total=%d", len(got.Problems), got.ProblemsTotal)
	}
	if empty := analyzePods(nil, t0); empty.Problems == nil {
		t.Error("an empty list must be [], not null")
	}
}

func TestWarningsByReason(t *testing.T) {
	ev := func(typ, reason string, age time.Duration) corev1.Event {
		return corev1.Event{Type: typ, Reason: reason, LastTimestamp: ago(age)}
	}
	total, top := warningsByReason([]corev1.Event{
		ev("Warning", "BackOff", time.Minute),
		ev("Warning", "BackOff", 5*time.Minute),
		ev("Warning", "FailedScheduling", 10*time.Minute),
		ev("Warning", "Old", 2*time.Hour),
		ev("Normal", "Pulled", time.Minute),
	}, t0)
	if total != 3 || len(top) != 2 || top[0] != (ReasonCount{"BackOff", 2}) {
		t.Errorf("total=%d top=%+v", total, top)
	}
}

func TestClusterRequests(t *testing.T) {
	node := corev1.Node{Status: corev1.NodeStatus{Allocatable: corev1.ResourceList{
		corev1.ResourceCPU: resource.MustParse("4"), corev1.ResourceMemory: resource.MustParse("8Gi")}}}
	withReq := func(name string, phase corev1.PodPhase, nodeName string) corev1.Pod {
		p := pod(name, phase, time.Hour)
		p.Spec.NodeName = nodeName
		p.Spec.Containers = []corev1.Container{{Resources: corev1.ResourceRequirements{Requests: corev1.ResourceList{
			corev1.ResourceCPU: resource.MustParse("500m"), corev1.ResourceMemory: resource.MustParse("1Gi")}}}}
		return p
	}
	r := clusterRequests([]corev1.Node{node, node}, []corev1.Pod{
		withReq("run", corev1.PodRunning, "n1"),
		withReq("unscheduled", corev1.PodPending, ""),
		withReq("done", corev1.PodSucceeded, "n1"),
	})
	if r.CPUAllocatable != 8 || r.CPURequested != 0.5 || r.MemoryRequested != float64(1<<30) {
		t.Errorf("%+v", r)
	}
}
