// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

func podWithPorts(name string, ports ...corev1.ContainerPort) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.PodSpec{Containers: []corev1.Container{{Name: "c", Ports: ports}}},
	}
}

func TestResolveTargetPort(t *testing.T) {
	pod := podWithPorts("p", corev1.ContainerPort{Name: "http", ContainerPort: 8080})

	cases := []struct {
		name    string
		sp      corev1.ServicePort
		want    int
		wantErr bool
	}{
		{"unset targetPort uses service port", corev1.ServicePort{Port: 80}, 80, false},
		{"numeric targetPort", corev1.ServicePort{Port: 80, TargetPort: intstr.FromInt32(9090)}, 9090, false},
		{"named targetPort", corev1.ServicePort{Port: 80, TargetPort: intstr.FromString("http")}, 8080, false},
		{"unknown named targetPort", corev1.ServicePort{Port: 80, TargetPort: intstr.FromString("grpc")}, 0, true},
	}
	for _, c := range cases {
		got, err := resolveTargetPort(&c.sp, pod)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err = %v, wantErr %v", c.name, err, c.wantErr)
			continue
		}
		if got != c.want {
			t.Errorf("%s: got %d, want %d", c.name, got, c.want)
		}
	}
}

func TestPickReadyPod(t *testing.T) {
	ready := func(name string, phase corev1.PodPhase, cond corev1.ConditionStatus) corev1.Pod {
		return corev1.Pod{
			ObjectMeta: metav1.ObjectMeta{Name: name},
			Status: corev1.PodStatus{
				Phase:      phase,
				Conditions: []corev1.PodCondition{{Type: corev1.PodReady, Status: cond}},
			},
		}
	}
	deleting := ready("deleting", corev1.PodRunning, corev1.ConditionTrue)
	now := metav1.Now()
	deleting.DeletionTimestamp = &now

	pods := []corev1.Pod{
		ready("pending", corev1.PodPending, corev1.ConditionFalse),
		ready("not-ready", corev1.PodRunning, corev1.ConditionFalse),
		deleting,
		ready("good", corev1.PodRunning, corev1.ConditionTrue),
	}
	got := pickReadyPod(pods)
	if got == nil || got.Name != "good" {
		t.Fatalf("pickReadyPod = %v, want pod 'good'", got)
	}
	if pickReadyPod(pods[:3]) != nil {
		t.Fatal("expected nil when no pod is ready")
	}
}

func TestStopPortForwardIsIdempotentPerSession(t *testing.T) {
	pfm := &PortForwardManager{sessions: map[string]*PortForwardSession{}}
	s := &PortForwardSession{ID: "x", stopChan: make(chan struct{})}
	pfm.sessions["x"] = s

	if err := pfm.StopPortForward("x"); err != nil {
		t.Fatal(err)
	}
	if err := pfm.StopPortForward("x"); err == nil {
		t.Fatal("second stop should report missing session")
	}
	s.stop() // повторное закрытие канала не должно паниковать
	select {
	case <-s.stopChan:
	default:
		t.Fatal("stopChan must be closed")
	}
}

func TestStopAllClearsSessions(t *testing.T) {
	pfm := &PortForwardManager{sessions: map[string]*PortForwardSession{}}
	for _, id := range []string{"a", "b"} {
		pfm.sessions[id] = &PortForwardSession{ID: id, stopChan: make(chan struct{})}
	}
	pfm.StopAll()
	if n := len(pfm.ListPortForwards()); n != 0 {
		t.Fatalf("expected no sessions, got %d", n)
	}
}

func TestSessionSnapshotKeepsClusterID(t *testing.T) {
	s := &PortForwardSession{ID: "x", ClusterID: "cluster-a", stopChan: make(chan struct{})}
	if got := s.snapshot(); got.ClusterID != "cluster-a" {
		t.Fatalf("snapshot ClusterID = %q, want cluster-a", got.ClusterID)
	}
}
