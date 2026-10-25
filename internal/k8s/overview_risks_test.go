package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func readyNode(name string, cpu, mem string, unschedulable bool) corev1.Node {
	return corev1.Node{
		ObjectMeta: metav1.ObjectMeta{Name: name},
		Spec:       corev1.NodeSpec{Unschedulable: unschedulable},
		Status: corev1.NodeStatus{
			Allocatable: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu), corev1.ResourceMemory: resource.MustParse(mem)},
			Conditions:  []corev1.NodeCondition{{Type: corev1.NodeReady, Status: corev1.ConditionTrue}},
		},
	}
}

func podOn(node string, phase corev1.PodPhase, cpu, mem string) corev1.Pod {
	return corev1.Pod{
		Spec: corev1.PodSpec{NodeName: node, Containers: []corev1.Container{{Resources: corev1.ResourceRequirements{
			Requests: corev1.ResourceList{corev1.ResourceCPU: resource.MustParse(cpu), corev1.ResourceMemory: resource.MustParse(mem)}}}}},
		Status: corev1.PodStatus{Phase: phase},
	}
}

func TestNodeHeadroom(t *testing.T) {
	got := nodeHeadroom(
		[]corev1.Node{readyNode("a", "4", "8Gi", false), readyNode("b", "4", "8Gi", true), readyNode("c", "2", "2Gi", false)},
		[]corev1.Pod{
			podOn("a", corev1.PodRunning, "1500m", "2Gi"),
			podOn("a", corev1.PodSucceeded, "1", "1Gi"), // завершённый не занимает места
			podOn("c", corev1.PodRunning, "4", "4Gi"),   // перебор: запас не уходит в минус
		},
	)
	if len(got) != 2 { // cordoned-узел b пропущен
		t.Fatalf("%+v", got)
	}
	if got[0].Name != "a" || got[0].CPUFree != 2.5 || got[0].MemoryFree != float64(6<<30) {
		t.Errorf("a: %+v", got[0])
	}
	if got[1].Name != "c" || got[1].CPUFree != 0 || got[1].MemoryFree != 0 {
		t.Errorf("c: %+v", got[1])
	}
}

func TestQuotaPressure(t *testing.T) {
	q := corev1.ResourceQuota{
		ObjectMeta: metav1.ObjectMeta{Namespace: "prod", Name: "q"},
		Status: corev1.ResourceQuotaStatus{
			Hard: corev1.ResourceList{"pods": resource.MustParse("10"), "requests.cpu": resource.MustParse("4"), "services": resource.MustParse("0")},
			Used: corev1.ResourceList{"pods": resource.MustParse("9"), "requests.cpu": resource.MustParse("1"), "services": resource.MustParse("0")},
		},
	}
	got, total := quotaPressure([]corev1.ResourceQuota{q})
	if total != 1 || len(got) != 1 || got[0].Resource != "pods" || got[0].Percent != 90 {
		t.Errorf("total=%d %+v", total, got)
	}
}

func TestPDBBlocks(t *testing.T) {
	mk := func(name string, allowed, expected int32) policyv1.PodDisruptionBudget {
		return policyv1.PodDisruptionBudget{
			ObjectMeta: metav1.ObjectMeta{Namespace: "ns", Name: name},
			Status:     policyv1.PodDisruptionBudgetStatus{DisruptionsAllowed: allowed, ExpectedPods: expected},
		}
	}
	got, total := pdbBlocks([]policyv1.PodDisruptionBudget{mk("blocked", 0, 3), mk("ok", 1, 3), mk("empty", 0, 0)})
	if total != 1 || got[0].Name != "blocked" {
		t.Errorf("total=%d %+v", total, got)
	}
}
