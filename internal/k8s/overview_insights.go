// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
)

// Сигналы для главного экрана, которые считаются только по данным API (informer-кеш):
// работают и без стека метрик.

const (
	restartWindow     = 24 * time.Hour
	warningWindow     = time.Hour
	pendingGrace      = 5 * time.Minute
	notReadyGrace     = 2 * time.Minute
	failedPodMaxAge   = 24 * time.Hour // старые Evicted/Failed-поды не считаем проблемой
	maxProblemPods    = 10
	maxWarningReasons = 5
)

// ProblemPod — под, которому нужно внимание.
type ProblemPod struct {
	Name      string `json:"name"`
	Namespace string `json:"namespace"`
	Reason    string `json:"reason"`
	Node      string `json:"node"`
	Restarts  int32  `json:"restarts"`
	Since     string `json:"since"` // RFC3339: относительное время считает фронтенд

	severity int
	at       time.Time
}

// ReasonCount — число Warning-событий с одной причиной.
type ReasonCount struct {
	Reason string `json:"reason"`
	Count  int    `json:"count"`
}

// ResourceRequests — сколько ресурсов зарезервировано requests'ами подов от allocatable узлов.
// CPU в ядрах, память в байтах.
type ResourceRequests struct {
	CPURequested      float64 `json:"cpuRequested"`
	CPUAllocatable    float64 `json:"cpuAllocatable"`
	MemoryRequested   float64 `json:"memoryRequested"`
	MemoryAllocatable float64 `json:"memoryAllocatable"`
}

type podInsights struct {
	Restarts      int // контейнеры, перезапускавшиеся за restartWindow
	OOMKilled     int // из них завершённые по OOMKilled
	Problems      []ProblemPod
	ProblemsTotal int
}

// waitingSeverity: причины Waiting, которые считаются проблемой (выше — серьёзнее).
var waitingSeverity = map[string]int{
	"CrashLoopBackOff":           3,
	"RunContainerError":          3,
	"ImagePullBackOff":           2,
	"ErrImagePull":               2,
	"InvalidImageName":           2,
	"CreateContainerConfigError": 2,
	"CreateContainerError":       2,
}

func analyzePods(pods []corev1.Pod, now time.Time) podInsights {
	out := podInsights{Problems: []ProblemPod{}}
	var all []ProblemPod
	for i := range pods {
		p := &pods[i]
		r, o := recentRestarts(p, now)
		out.Restarts += r
		out.OOMKilled += o
		if pp, ok := podProblem(p, now); ok {
			all = append(all, pp)
		}
	}
	sort.SliceStable(all, func(i, j int) bool {
		a, b := all[i], all[j]
		if a.severity != b.severity {
			return a.severity > b.severity
		}
		if !a.at.Equal(b.at) {
			return a.at.Before(b.at) // давно болеющие выше
		}
		if a.Namespace != b.Namespace {
			return a.Namespace < b.Namespace
		}
		return a.Name < b.Name
	})
	out.ProblemsTotal = len(all)
	if len(all) > maxProblemPods {
		all = all[:maxProblemPods]
	}
	if len(all) > 0 {
		out.Problems = all
	}
	return out
}

// recentRestarts считает контейнеры пода, чей прошлый запуск завершился в пределах restartWindow.
// Это число перезапустившихся контейнеров, а не сумма restartCount: у контейнера в crash loop
// счётчик растёт каждые минуты и перекрыл бы все остальные сигналы.
func recentRestarts(p *corev1.Pod, now time.Time) (restarts, oom int) {
	if p.DeletionTimestamp != nil {
		return 0, 0
	}
	for _, cs := range p.Status.ContainerStatuses {
		t := cs.LastTerminationState.Terminated
		if t == nil || t.FinishedAt.IsZero() || now.Sub(t.FinishedAt.Time) > restartWindow {
			continue
		}
		restarts++
		if t.Reason == "OOMKilled" {
			oom++
		}
	}
	return
}

func podProblem(p *corev1.Pod, now time.Time) (ProblemPod, bool) {
	if p.DeletionTimestamp != nil || p.Status.Phase == corev1.PodSucceeded {
		return ProblemPod{}, false
	}
	pp := ProblemPod{Name: p.Name, Namespace: p.Namespace, Node: p.Spec.NodeName}
	for _, cs := range p.Status.ContainerStatuses {
		pp.Restarts += cs.RestartCount
	}
	pp.at = podConditionSince(p)
	set := func(reason string, sev int) (ProblemPod, bool) {
		pp.Reason, pp.severity = reason, sev
		pp.Since = pp.at.UTC().Format(time.RFC3339)
		return pp, true
	}

	if p.Status.Phase == corev1.PodFailed {
		if now.Sub(p.CreationTimestamp.Time) > failedPodMaxAge {
			return ProblemPod{}, false
		}
		reason := p.Status.Reason
		if reason == "" {
			reason = "Failed"
		}
		return set(reason, 3)
	}

	statuses := append(append([]corev1.ContainerStatus{}, p.Status.InitContainerStatuses...), p.Status.ContainerStatuses...)
	best, bestSev := "", 0
	for _, cs := range statuses {
		if w := cs.State.Waiting; w != nil {
			if sev := waitingSeverity[w.Reason]; sev > bestSev {
				best, bestSev = w.Reason, sev
			}
		}
	}
	if bestSev > 0 {
		return set(best, bestSev)
	}

	switch p.Status.Phase {
	case corev1.PodPending:
		if now.Sub(p.CreationTimestamp.Time) <= pendingGrace {
			return ProblemPod{}, false
		}
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodScheduled && c.Status == corev1.ConditionFalse && c.Reason == corev1.PodReasonUnschedulable {
				return set("Unschedulable", 1)
			}
		}
		return set("Pending", 1)
	case corev1.PodRunning:
		for _, c := range p.Status.Conditions {
			if c.Type == corev1.PodReady && c.Status != corev1.ConditionTrue && now.Sub(c.LastTransitionTime.Time) > notReadyGrace {
				return set("NotReady", 1)
			}
		}
	}
	return ProblemPod{}, false
}

// podConditionSince: с какого момента под в текущем состоянии (Ready/Scheduled), иначе время создания.
func podConditionSince(p *corev1.Pod) time.Time {
	for _, c := range p.Status.Conditions {
		if (c.Type == corev1.PodReady || c.Type == corev1.PodScheduled) && c.Status != corev1.ConditionTrue && !c.LastTransitionTime.IsZero() {
			return c.LastTransitionTime.Time
		}
	}
	return p.CreationTimestamp.Time
}

// warningsByReason: число Warning-событий за warningWindow и топ причин.
func warningsByReason(events []corev1.Event, now time.Time) (int, []ReasonCount) {
	counts := map[string]int{}
	total := 0
	for i := range events {
		ev := &events[i]
		if ev.Type != corev1.EventTypeWarning || now.Sub(eventTimestamp(ev)) > warningWindow {
			continue
		}
		total++
		reason := ev.Reason
		if reason == "" {
			reason = "Unknown"
		}
		counts[reason]++
	}
	out := make([]ReasonCount, 0, len(counts))
	for r, n := range counts {
		out = append(out, ReasonCount{Reason: r, Count: n})
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].Count != out[j].Count {
			return out[i].Count > out[j].Count
		}
		return out[i].Reason < out[j].Reason
	})
	if len(out) > maxWarningReasons {
		out = out[:maxWarningReasons]
	}
	return total, out
}

// clusterRequests суммирует allocatable узлов и requests запланированных подов.
// Под считается как сумма requests его контейнеров (init-контейнеры и overhead не учитываются).
func clusterRequests(nodes []corev1.Node, pods []corev1.Pod) ResourceRequests {
	var r ResourceRequests
	for i := range nodes {
		a := nodes[i].Status.Allocatable
		r.CPUAllocatable += float64(a.Cpu().MilliValue()) / 1000
		r.MemoryAllocatable += float64(a.Memory().Value())
	}
	for i := range pods {
		p := &pods[i]
		if p.Spec.NodeName == "" || (p.Status.Phase != corev1.PodRunning && p.Status.Phase != corev1.PodPending) {
			continue
		}
		for _, c := range p.Spec.Containers {
			r.CPURequested += float64(c.Resources.Requests.Cpu().MilliValue()) / 1000
			r.MemoryRequested += float64(c.Resources.Requests.Memory().Value())
		}
	}
	return r
}
