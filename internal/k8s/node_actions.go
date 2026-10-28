// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"context"
	"fmt"
	"sort"
	"strings"
	"sync"
	"time"

	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/utils/ptr"
)

const (
	// drainTimeout — сколько ждём, пока все поды покинут узел.
	drainTimeout = 3 * time.Minute

	// Shell на узле: временный привилегированный под с nsenter в пространства имён PID 1 хоста.
	nodeShellNamespace    = "kube-system"
	nodeShellPrefix       = "kubron-node-shell-"
	nodeShellLabel        = "kubron.io/node-shell"
	nodeShellImage        = "docker.io/alpine:3.19" // в busybox есть nsenter
	nodeShellStartTimeout = 60 * time.Second
	// Страховка на случай, если приложение закрыли, не дав подчистить под
	nodeShellMaxLifetime = int64(4 * 60 * 60)

	mirrorPodAnnotation = "kubernetes.io/config.mirror"
)

// ─── Cordon / Uncordon ─────────────────────────────────────

// CordonNode помечает узел как неназначаемый (cordon=true) или возвращает его в работу (false).
func (m *Manager) CordonNode(name string, cordon bool) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return setUnschedulable(ctx, conn, name, cordon)
}

func setUnschedulable(ctx context.Context, conn *ClusterConnection, name string, v bool) error {
	patch := fmt.Sprintf(`{"spec":{"unschedulable":%t}}`, v)
	if _, err := conn.Client.CoreV1().Nodes().Patch(ctx, name, types.MergePatchType, []byte(patch), metav1.PatchOptions{}); err != nil {
		return fmt.Errorf("failed to update node %s: %w", name, err)
	}
	return nil
}

// ─── Drain ─────────────────────────────────────────────────

// DrainNode закрывает узел для планирования и эвакуирует с него поды через Eviction API
// (PodDisruptionBudget соблюдаются). Как и kubectl drain, пропускает поды DaemonSet и
// mirror-поды. Поды без контроллера не пересоздадутся, поэтому их эвакуация отклоняется.
func (m *Manager) DrainNode(name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), drainTimeout)
	defer cancel()

	if err := setUnschedulable(ctx, conn, name, true); err != nil {
		return err
	}

	list, err := conn.Client.CoreV1().Pods(metav1.NamespaceAll).List(ctx, metav1.ListOptions{
		FieldSelector: "spec.nodeName=" + name,
	})
	if err != nil {
		return fmt.Errorf("failed to get pods of node %s: %w", name, err)
	}

	var evict []corev1.Pod
	var orphans []string
	for i := range list.Items {
		p := list.Items[i]
		switch {
		case isMirrorPod(&p), isDaemonSetPod(&p):
			// управляются kubelet / DaemonSet-контроллером: эвакуировать бессмысленно
		case metav1.GetControllerOf(&p) == nil && !isPodFinished(&p):
			orphans = append(orphans, p.Namespace+"/"+p.Name)
		default:
			evict = append(evict, p)
		}
	}
	if len(orphans) > 0 {
		sort.Strings(orphans)
		return fmt.Errorf("node %s is cordoned but not drained: pods without a controller will not be recreated (%s)",
			name, summarize(orphans))
	}

	var (
		wg     sync.WaitGroup
		mu     sync.Mutex
		failed []string
	)
	for _, p := range evict {
		wg.Add(1)
		go func(p corev1.Pod) {
			defer wg.Done()
			if err := evictAndWait(ctx, conn, &p); err != nil {
				mu.Lock()
				failed = append(failed, fmt.Sprintf("%s/%s: %v", p.Namespace, p.Name, err))
				mu.Unlock()
			}
		}(p)
	}
	wg.Wait()

	if len(failed) > 0 {
		sort.Strings(failed)
		return fmt.Errorf("failed to evict %d of %d pods: %s", len(failed), len(evict), summarize(failed))
	}
	return nil
}

// evictAndWait просит эвакуировать под и ждёт, пока он исчезнет. 429 от Eviction API означает,
// что эвакуацию пока запрещает PodDisruptionBudget: повторяем, пока не истечёт ctx.
func evictAndWait(ctx context.Context, conn *ClusterConnection, pod *corev1.Pod) error {
	ev := &policyv1.Eviction{ObjectMeta: metav1.ObjectMeta{Name: pod.Name, Namespace: pod.Namespace}}
	for {
		err := conn.Client.PolicyV1().Evictions(pod.Namespace).Evict(ctx, ev)
		if err == nil || apierrors.IsNotFound(err) {
			break
		}
		if !apierrors.IsTooManyRequests(err) {
			return err
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("eviction is blocked by a PodDisruptionBudget")
		case <-time.After(5 * time.Second):
		}
	}

	pods := conn.Client.CoreV1().Pods(pod.Namespace)
	for {
		cur, err := pods.Get(ctx, pod.Name, metav1.GetOptions{})
		// UID сменился — под пересоздан контроллером (StatefulSet) уже на другом месте
		if apierrors.IsNotFound(err) || (err == nil && cur.UID != pod.UID) {
			return nil
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("pod did not terminate within the allotted time")
		case <-time.After(time.Second):
		}
	}
}

func isMirrorPod(p *corev1.Pod) bool {
	_, ok := p.Annotations[mirrorPodAnnotation]
	return ok
}

func isDaemonSetPod(p *corev1.Pod) bool {
	ref := metav1.GetControllerOf(p)
	return ref != nil && ref.Kind == "DaemonSet"
}

func isPodFinished(p *corev1.Pod) bool {
	return p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed
}

// summarize показывает первые элементы списка, чтобы сообщение об ошибке не разрасталось.
func summarize(items []string) string {
	const max = 3
	if len(items) <= max {
		return strings.Join(items, "; ")
	}
	return fmt.Sprintf("%s; and %d more", strings.Join(items[:max], "; "), len(items)-max)
}

// ─── Shell на узле ─────────────────────────────────────────

// NodeShellSession — временный под, к которому интерфейс подключается в режиме attach.
type NodeShellSession struct {
	Namespace string `json:"namespace"`
	Pod       string `json:"pod"`
	Container string `json:"container"`
}

// StartNodeShell создаёт на узле привилегированный под (hostPID) с root-оболочкой хоста
// и ждёт его запуска. Под удаляется, когда завершается терминальная сессия
// (см. StartTerminalSession), а activeDeadlineSeconds — страховка на случай аварийного выхода.
func (m *Manager) StartNodeShell(node string) (*NodeShellSession, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), nodeShellStartTimeout)
	defer cancel()

	if _, err := conn.Client.CoreV1().Nodes().Get(ctx, node, metav1.GetOptions{}); err != nil {
		return nil, fmt.Errorf("node %s: %w", node, err)
	}

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			GenerateName: nodeShellPrefix,
			Namespace:    nodeShellNamespace,
			Labels: map[string]string{
				nodeShellLabel:                 "true",
				"app.kubernetes.io/managed-by": "kubron",
			},
			Annotations: map[string]string{"kubron.io/node": node},
		},
		Spec: corev1.PodSpec{
			NodeName:                      node, // в обход планировщика: работает и на cordoned-узле
			HostPID:                       true,
			HostNetwork:                   true,
			RestartPolicy:                 corev1.RestartPolicyNever,
			ActiveDeadlineSeconds:         ptr.To(nodeShellMaxLifetime),
			TerminationGracePeriodSeconds: ptr.To(int64(0)),
			AutomountServiceAccountToken:  ptr.To(false),
			Tolerations:                   []corev1.Toleration{{Operator: corev1.TolerationOpExists}},
			Containers: []corev1.Container{{
				Name:            "shell",
				Image:           nodeShellImage,
				ImagePullPolicy: corev1.PullIfNotPresent,
				Stdin:           true,
				TTY:             true,
				SecurityContext: &corev1.SecurityContext{Privileged: ptr.To(true)},
				// Оболочка хоста: bash, если есть, иначе sh
				Command: []string{
					"nsenter", "-t", "1", "-m", "-u", "-i", "-n", "-p", "--",
					"sh", "-c", "if command -v bash >/dev/null 2>&1; then exec bash; else exec sh; fi",
				},
			}},
		},
	}

	created, err := conn.Client.CoreV1().Pods(nodeShellNamespace).Create(ctx, pod, metav1.CreateOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to create shell pod on node %s: %w", node, err)
	}
	fail := func(err error) (*NodeShellSession, error) {
		deleteNodeShellPod(conn, nodeShellNamespace, created.Name)
		return nil, err
	}

	for {
		p, err := conn.Client.CoreV1().Pods(nodeShellNamespace).Get(ctx, created.Name, metav1.GetOptions{})
		if err == nil {
			switch p.Status.Phase {
			case corev1.PodRunning:
				return &NodeShellSession{Namespace: nodeShellNamespace, Pod: created.Name, Container: "shell"}, nil
			case corev1.PodFailed, corev1.PodSucceeded:
				return fail(fmt.Errorf("shell pod terminated before starting"))
			}
			for _, cs := range p.Status.ContainerStatuses {
				if w := cs.State.Waiting; w != nil {
					switch w.Reason {
					case "ErrImagePull", "ImagePullBackOff", "InvalidImageName", "CreateContainerConfigError":
						return fail(fmt.Errorf("shell pod failed to start (%s): %s", w.Reason, w.Message))
					}
				}
			}
		}
		select {
		case <-ctx.Done():
			return fail(fmt.Errorf("timed out starting shell on node %s (%s)", node, nodeShellStartTimeout))
		case <-time.After(500 * time.Millisecond):
		}
	}
}

// deleteNodeShellPod удаляет под node shell. Чужие поды не трогает: проверяет имя, namespace и метку.
// Безопасен для повторного вызова.
func deleteNodeShellPod(conn *ClusterConnection, namespace, name string) {
	if namespace != nodeShellNamespace || !strings.HasPrefix(name, nodeShellPrefix) {
		return
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	p, err := conn.Client.CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil || p.Labels[nodeShellLabel] != "true" {
		return
	}
	_ = conn.Client.CoreV1().Pods(namespace).Delete(ctx, name, metav1.DeleteOptions{
		GracePeriodSeconds: ptr.To(int64(0)),
	})
}
