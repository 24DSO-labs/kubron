package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/robfig/cron/v3"
	batchv1 "k8s.io/api/batch/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// === StatefulSets ===

type StatefulSetInfo struct {
	Name          string   `json:"name"`
	Namespace     string   `json:"namespace"`
	Replicas      int32    `json:"replicas"`
	ReadyReplicas int32    `json:"readyReplicas"`
	Age           string   `json:"age"`
	Images        []string `json:"images"`
	ServiceName   string   `json:"serviceName"`

	Selector        string               `json:"selector"`
	Strategy        string               `json:"strategy"`
	NodeSelector    string               `json:"nodeSelector"`
	Labels          map[string]string    `json:"labels"`
	Annotations     map[string]string    `json:"annotations"`
	Tolerations     []TolerationInfo     `json:"tolerations"`
	Affinity        *AffinityInfo        `json:"affinity"`
	PodAntiAffinity *PodAntiAffinityInfo `json:"podAntiAffinity"`
}

func (m *Manager) GetStatefulSets(namespace string) ([]StatefulSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.AppsV1().StatefulSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []StatefulSetInfo
	for _, s := range list.Items {
		var images []string
		for _, c := range s.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		var replicas int32 = 1
		if s.Spec.Replicas != nil {
			replicas = *s.Spec.Replicas
		}
		var selector string
		if s.Spec.Selector != nil && len(s.Spec.Selector.MatchLabels) > 0 {
			first := true
			for k, v := range s.Spec.Selector.MatchLabels {
				if !first {
					selector += ", "
				}
				selector += k + "=" + v
				first = false
			}
		}

		// Convert tolerations
		var tolerations []TolerationInfo
		for _, tol := range s.Spec.Template.Spec.Tolerations {
			tolInfo := TolerationInfo{
				Key:      tol.Key,
				Operator: string(tol.Operator),
				Value:    tol.Value,
				Effect:   string(tol.Effect),
			}
			tolerations = append(tolerations, tolInfo)
		}

		// Extract node affinity information
		var affinity *AffinityInfo
		if s.Spec.Template.Spec.Affinity != nil && s.Spec.Template.Spec.Affinity.NodeAffinity != nil {
			na := s.Spec.Template.Spec.Affinity.NodeAffinity
			affinity = &AffinityInfo{
				NodeAffinity: &NodeAffinityInfo{},
			}

			if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				terms := na.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
				selectorTerms := make([]NodeSelectorTerm, 0)
				for _, term := range terms {
					matchExpressions := make([]MatchExpression, 0)
					for _, expr := range term.MatchExpressions {
						matchExpressions = append(matchExpressions, MatchExpression{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					selectorTerms = append(selectorTerms, NodeSelectorTerm{
						MatchExpressions: matchExpressions,
					})
				}
				affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &NodeSelectorInfo{
					NodeSelectorTerms: selectorTerms,
				}
			}

			if na.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				preferred := make([]PreferredNodeSelector, 0)
				for _, pref := range na.PreferredDuringSchedulingIgnoredDuringExecution {
					matchExpressions := make([]MatchExpression, 0)
					for _, expr := range pref.Preference.MatchExpressions {
						matchExpressions = append(matchExpressions, MatchExpression{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					preferred = append(preferred, PreferredNodeSelector{
						Weight: pref.Weight,
						Preference: NodeSelectorTerm{
							MatchExpressions: matchExpressions,
						},
					})
				}
				affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferred
			}
		}

		// Build node selector string
		nodeSelectorStr := ""
		if len(s.Spec.Template.Spec.NodeSelector) > 0 {
			first := true
			for k, v := range s.Spec.Template.Spec.NodeSelector {
				if !first {
					nodeSelectorStr += ", "
				}
				nodeSelectorStr += k + "=" + v
				first = false
			}
		}

		// Build pod anti-affinity info
		var podAntiAffinity *PodAntiAffinityInfo
		if s.Spec.Template.Spec.Affinity != nil && s.Spec.Template.Spec.Affinity.PodAntiAffinity != nil {
			paa := s.Spec.Template.Spec.Affinity.PodAntiAffinity
			podAntiAffinity = &PodAntiAffinityInfo{}

			// Preferred affinities
			if paa.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				for _, pref := range paa.PreferredDuringSchedulingIgnoredDuringExecution {
					term := PodAffinityTermInfo{
						TopologyKey: pref.PodAffinityTerm.TopologyKey,
						Namespaces:  pref.PodAffinityTerm.Namespaces,
					}
					if pref.PodAffinityTerm.LabelSelector != nil {
						term.LabelSelector = &LabelSelectorInfo{
							MatchLabels: pref.PodAffinityTerm.LabelSelector.MatchLabels,
						}
					}
					podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution = append(
						podAntiAffinity.PreferredDuringSchedulingIgnoredDuringExecution,
						PreferredAffinityTermInfo{
							Weight:          pref.Weight,
							PodAffinityTerm: term,
						},
					)
				}
			}

			// Required affinities
			if paa.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				for _, req := range paa.RequiredDuringSchedulingIgnoredDuringExecution {
					term := PodAffinityTermInfo{
						TopologyKey: req.TopologyKey,
						Namespaces:  req.Namespaces,
					}
					if req.LabelSelector != nil {
						term.LabelSelector = &LabelSelectorInfo{
							MatchLabels: req.LabelSelector.MatchLabels,
						}
					}
					podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution = append(
						podAntiAffinity.RequiredDuringSchedulingIgnoredDuringExecution,
						RequiredAffinityTermInfo{
							PodAffinityTerm: term,
						},
					)
				}
			}
		}

		items = append(items, StatefulSetInfo{
			Name:            s.Name,
			Namespace:       s.Namespace,
			Replicas:        replicas,
			ReadyReplicas:   s.Status.ReadyReplicas,
			Age:             formatAge(s.CreationTimestamp.Time),
			Images:          images,
			ServiceName:     s.Spec.ServiceName,
			Selector:        selector,
			Strategy:        string(s.Spec.UpdateStrategy.Type),
			NodeSelector:    nodeSelectorStr,
			Labels:          s.Labels,
			Annotations:     s.Annotations,
			Tolerations:     tolerations,
			Affinity:        affinity,
			PodAntiAffinity: podAntiAffinity,
		})
	}
	return items, nil
}

func (m *Manager) DeleteStatefulSet(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	return conn.Client.AppsV1().StatefulSets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

func (m *Manager) RestartStatefulSet(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	sts, err := conn.Client.AppsV1().StatefulSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if sts.Spec.Template.Annotations == nil {
		sts.Spec.Template.Annotations = make(map[string]string)
	}
	sts.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	_, err = conn.Client.AppsV1().StatefulSets(namespace).Update(ctx, sts, metav1.UpdateOptions{})
	return err
}

// === DaemonSets ===

type NodeAffinityInfo struct {
	RequiredDuringSchedulingIgnoredDuringExecution  *NodeSelectorInfo       `json:"requiredDuringSchedulingIgnoredDuringExecution"`
	PreferredDuringSchedulingIgnoredDuringExecution []PreferredNodeSelector `json:"preferredDuringSchedulingIgnoredDuringExecution"`
}

type NodeSelectorInfo struct {
	NodeSelectorTerms []NodeSelectorTerm `json:"nodeSelectorTerms"`
}

type NodeSelectorTerm struct {
	MatchExpressions []MatchExpression `json:"matchExpressions"`
}

type PreferredNodeSelector struct {
	Weight     int32            `json:"weight"`
	Preference NodeSelectorTerm `json:"preference"`
}

type MatchExpression struct {
	Key      string   `json:"key"`
	Operator string   `json:"operator"`
	Values   []string `json:"values"`
}

type AffinityInfo struct {
	NodeAffinity *NodeAffinityInfo `json:"nodeAffinity"`
}

type DaemonSetInfo struct {
	Name                   string            `json:"name"`
	Namespace              string            `json:"namespace"`
	Desired                int32             `json:"desired"`
	Current                int32             `json:"current"`
	Ready                  int32             `json:"ready"`
	Available              int32             `json:"available"`
	DesiredNumberScheduled int32             `json:"desiredNumberScheduled"`
	CurrentNumberScheduled int32             `json:"currentNumberScheduled"`
	NumberReady            int32             `json:"numberReady"`
	NumberAvailable        int32             `json:"numberAvailable"`
	NumberUpdatedScheduled int32             `json:"updated"`
	Age                    string            `json:"age"`
	Created                string            `json:"created"`
	Images                 []string          `json:"images"`
	Labels                 map[string]string `json:"labels"`
	Annotations            map[string]string `json:"annotations"`
	Selector               string            `json:"selector"`
	NodeSelector           string            `json:"nodeSelector"`
	Strategy               string            `json:"strategy"`
	Conditions             []ConditionInfo   `json:"conditions"`
	Tolerations            []TolerationInfo  `json:"tolerations"`
	Affinity               *AffinityInfo     `json:"affinity"`
}

func (m *Manager) GetDaemonSets(namespace string) ([]DaemonSetInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.AppsV1().DaemonSets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []DaemonSetInfo
	for _, d := range list.Items {
		var images []string
		for _, c := range d.Spec.Template.Spec.Containers {
			images = append(images, c.Image)
		}
		ns := ""
		if d.Spec.Template.Spec.NodeSelector != nil {
			var parts []string
			for k, v := range d.Spec.Template.Spec.NodeSelector {
				parts = append(parts, fmt.Sprintf("%s=%s", k, v))
			}
			ns = strings.Join(parts, ", ")
		}

		// Build selector string from MatchLabels
		selectorStr := ""
		if d.Spec.Selector != nil && len(d.Spec.Selector.MatchLabels) > 0 {
			first := true
			for k, v := range d.Spec.Selector.MatchLabels {
				if !first {
					selectorStr += ", "
				}
				selectorStr += k + "=" + v
				first = false
			}
		}

		// Convert conditions
		var conditions []ConditionInfo
		for _, cond := range d.Status.Conditions {
			conditions = append(conditions, ConditionInfo{
				Type:               string(cond.Type),
				Status:             string(cond.Status),
				LastTransitionTime: cond.LastTransitionTime.Format(time.RFC3339),
				Reason:             cond.Reason,
				Message:            cond.Message,
			})
		}

		// Convert tolerations
		var tolerations []TolerationInfo
		for _, tol := range d.Spec.Template.Spec.Tolerations {
			tolInfo := TolerationInfo{
				Key:      tol.Key,
				Operator: string(tol.Operator),
				Value:    tol.Value,
				Effect:   string(tol.Effect),
			}
			tolerations = append(tolerations, tolInfo)
		}

		// Extract node affinity information
		var affinity *AffinityInfo
		if d.Spec.Template.Spec.Affinity != nil && d.Spec.Template.Spec.Affinity.NodeAffinity != nil {
			na := d.Spec.Template.Spec.Affinity.NodeAffinity
			affinity = &AffinityInfo{
				NodeAffinity: &NodeAffinityInfo{},
			}

			// Process required node affinity
			if na.RequiredDuringSchedulingIgnoredDuringExecution != nil {
				terms := na.RequiredDuringSchedulingIgnoredDuringExecution.NodeSelectorTerms
				selectorTerms := make([]NodeSelectorTerm, 0)
				for _, term := range terms {
					matchExpressions := make([]MatchExpression, 0)
					for _, expr := range term.MatchExpressions {
						matchExpressions = append(matchExpressions, MatchExpression{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					selectorTerms = append(selectorTerms, NodeSelectorTerm{
						MatchExpressions: matchExpressions,
					})
				}
				affinity.NodeAffinity.RequiredDuringSchedulingIgnoredDuringExecution = &NodeSelectorInfo{
					NodeSelectorTerms: selectorTerms,
				}
			}

			// Process preferred node affinity
			if na.PreferredDuringSchedulingIgnoredDuringExecution != nil {
				preferred := make([]PreferredNodeSelector, 0)
				for _, pref := range na.PreferredDuringSchedulingIgnoredDuringExecution {
					matchExpressions := make([]MatchExpression, 0)
					for _, expr := range pref.Preference.MatchExpressions {
						matchExpressions = append(matchExpressions, MatchExpression{
							Key:      expr.Key,
							Operator: string(expr.Operator),
							Values:   expr.Values,
						})
					}
					preferred = append(preferred, PreferredNodeSelector{
						Weight: pref.Weight,
						Preference: NodeSelectorTerm{
							MatchExpressions: matchExpressions,
						},
					})
				}
				affinity.NodeAffinity.PreferredDuringSchedulingIgnoredDuringExecution = preferred
			}
		}

		items = append(items, DaemonSetInfo{
			Name:                   d.Name,
			Namespace:              d.Namespace,
			Desired:                d.Status.DesiredNumberScheduled,
			Current:                d.Status.CurrentNumberScheduled,
			Ready:                  d.Status.NumberReady,
			Available:              d.Status.NumberAvailable,
			DesiredNumberScheduled: d.Status.DesiredNumberScheduled,
			CurrentNumberScheduled: d.Status.CurrentNumberScheduled,
			NumberReady:            d.Status.NumberReady,
			NumberAvailable:        d.Status.NumberAvailable,
			NumberUpdatedScheduled: d.Status.UpdatedNumberScheduled,
			Age:                    formatAge(d.CreationTimestamp.Time),
			Created:                d.CreationTimestamp.Format(time.RFC3339),
			Images:                 images,
			Labels:                 d.Labels,
			Annotations:            d.Annotations,
			Selector:               selectorStr,
			NodeSelector:           ns,
			Strategy:               string(d.Spec.UpdateStrategy.Type),
			Conditions:             conditions,
			Tolerations:            tolerations,
			Affinity:               affinity,
		})
	}
	return items, nil
}

// === Jobs ===

type JobInfo struct {
	Name        string   `json:"name"`
	Namespace   string   `json:"namespace"`
	Completions string   `json:"completions"`
	Succeeded   int32    `json:"succeeded"`
	Failed      int32    `json:"failed"`
	Active      int32    `json:"active"`
	Duration    string   `json:"duration"`
	Age         string   `json:"age"`
	Status      string   `json:"status"`
	Conditions  []string `json:"conditions"`
}

type jobWarn struct {
	count int
	msg   string
}

func (m *Manager) GetJobs(namespace string) ([]JobInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []JobInfo
	for _, j := range list.Items {
		completions := "1"
		if j.Spec.Completions != nil {
			completions = fmt.Sprintf("%d", *j.Spec.Completions)
		}

		conds := make([]string, 0, len(j.Status.Conditions))
		var failed, complete, suspended bool
		for _, c := range j.Status.Conditions {
			if string(c.Status) != "True" {
				continue
			}
			conds = append(conds, string(c.Type))
			switch c.Type {
			case batchv1.JobFailed:
				failed = true
			case batchv1.JobComplete:
				complete = true
			case batchv1.JobSuspended:
				suspended = true
			}
		}
		status := "Running"
		switch {
		case failed:
			status = "Failed"
		case complete:
			status = "Complete"
		case suspended:
			status = "Suspended"
		}

		duration := ""
		if j.Status.StartTime != nil {
			end := time.Now()
			if j.Status.CompletionTime != nil {
				end = j.Status.CompletionTime.Time
			}
			// formatAge форматирует «сколько прошло с момента t»: сдвигаем t на длительность
			duration = formatAge(time.Now().Add(-end.Sub(j.Status.StartTime.Time)))
		}

		items = append(items, JobInfo{
			Name:        j.Name,
			Namespace:   j.Namespace,
			Completions: fmt.Sprintf("%d/%s", j.Status.Succeeded, completions),
			Succeeded:   j.Status.Succeeded,
			Failed:      j.Status.Failed,
			Active:      j.Status.Active,
			Duration:    duration,
			Age:         formatAge(j.CreationTimestamp.Time),
			Status:      status,
			Conditions:  conds,
		})
	}
	return items, nil
}

func (m *Manager) DeleteJob(namespace, jobName string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	policy := metav1.DeletePropagationBackground
	return conn.Client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{PropagationPolicy: &policy})
}

func (m *Manager) DeleteDaemonSet(namespace, daemonSetName string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return conn.Client.AppsV1().DaemonSets(namespace).Delete(ctx, daemonSetName, metav1.DeleteOptions{})
}

// RestartDaemonSet триггерит rolling restart: меняет аннотацию в шаблоне пода (как kubectl rollout restart)
func (m *Manager) RestartDaemonSet(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	ds, err := conn.Client.AppsV1().DaemonSets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		return err
	}
	if ds.Spec.Template.Annotations == nil {
		ds.Spec.Template.Annotations = make(map[string]string)
	}
	ds.Spec.Template.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)
	_, err = conn.Client.AppsV1().DaemonSets(namespace).Update(ctx, ds, metav1.UpdateOptions{})
	return err
}

// RetryJob creates a new job from an existing job configuration
func (m *Manager) RetryJob(namespace, jobName string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	// Get the existing job
	job, err := conn.Client.BatchV1().Jobs(namespace).Get(ctx, jobName, metav1.GetOptions{})
	if err != nil {
		return err
	}

	// Delete the old job first
	propagationPolicy := metav1.DeletePropagationBackground
	err = conn.Client.BatchV1().Jobs(namespace).Delete(ctx, jobName, metav1.DeleteOptions{
		PropagationPolicy: &propagationPolicy,
	})
	if err != nil {
		return err
	}

	// Clear resource version and uid for the new job
	job.ResourceVersion = ""
	job.UID = ""
	job.Status = batchv1.JobStatus{}

	// Add annotation to track retry
	if job.Annotations == nil {
		job.Annotations = make(map[string]string)
	}
	job.Annotations["kubectl.kubernetes.io/restartedAt"] = time.Now().Format(time.RFC3339)

	// Create the new job
	_, err = conn.Client.BatchV1().Jobs(namespace).Create(ctx, job, metav1.CreateOptions{})
	return err
}

// === CronJobs ===

type CronJobInfo struct {
	Name          string `json:"name"`
	Namespace     string `json:"namespace"`
	Schedule      string `json:"schedule"`
	Suspend       bool   `json:"suspend"`
	Active        int    `json:"active"`
	LastSchedule  string `json:"lastSchedule"`
	NextExecution string `json:"nextExecution"`
	TimeZone      string `json:"timeZone"`
	Age           string `json:"age"`
}

func (m *Manager) GetCronJobs(namespace string) ([]CronJobInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.BatchV1().CronJobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []CronJobInfo
	for _, cj := range list.Items {
		lastSchedule := "—"
		if cj.Status.LastScheduleTime != nil {
			lastSchedule = formatAge(cj.Status.LastScheduleTime.Time)
		}
		suspend := false
		if cj.Spec.Suspend != nil {
			suspend = *cj.Spec.Suspend
		}
		// Вычисляем следующее время выполнения
		nextExecution := "—"
		if !suspend && cj.Spec.Schedule != "" {
			nextExecution = computeNextExecution(cj.Spec.Schedule, cj.Spec.TimeZone)
		}

		// Временная зона
		timeZone := "—"
		if cj.Spec.TimeZone != nil && *cj.Spec.TimeZone != "" {
			timeZone = *cj.Spec.TimeZone
		}

		items = append(items, CronJobInfo{
			Name:          cj.Name,
			Namespace:     cj.Namespace,
			Schedule:      cj.Spec.Schedule,
			Suspend:       suspend,
			Active:        len(cj.Status.Active),
			LastSchedule:  lastSchedule,
			NextExecution: nextExecution,
			TimeZone:      timeZone,
			Age:           formatAge(cj.CreationTimestamp.Time),
		})
	}
	return items, nil
}

func (m *Manager) DeleteCronJob(namespace, name string) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	return conn.Client.BatchV1().CronJobs(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}

// GetCronJobJobs возвращает Job'ы, созданные указанным CronJob (по ownerReference).
func (m *Manager) GetCronJobJobs(namespace, cronJobName string) ([]JobInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.BatchV1().Jobs(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []JobInfo
	for _, j := range list.Items {
		// Проверяем ownerReference — Job должен принадлежать указанному CronJob
		isOwnedByCronJob := false
		for _, ref := range j.OwnerReferences {
			if ref.Kind == "CronJob" && ref.Name == cronJobName {
				isOwnedByCronJob = true
				break
			}
		}
		if !isOwnedByCronJob {
			continue
		}

		completions := "1"
		if j.Spec.Completions != nil {
			completions = fmt.Sprintf("%d", *j.Spec.Completions)
		}

		conds := make([]string, 0, len(j.Status.Conditions))
		var failed, complete, suspended bool
		for _, c := range j.Status.Conditions {
			if string(c.Status) != "True" {
				continue
			}
			conds = append(conds, string(c.Type))
			switch c.Type {
			case batchv1.JobFailed:
				failed = true
			case batchv1.JobComplete:
				complete = true
			case batchv1.JobSuspended:
				suspended = true
			}
		}
		status := "Running"
		switch {
		case failed:
			status = "Failed"
		case complete:
			status = "Complete"
		case suspended:
			status = "Suspended"
		}

		duration := ""
		if j.Status.StartTime != nil {
			end := time.Now()
			if j.Status.CompletionTime != nil {
				end = j.Status.CompletionTime.Time
			}
			duration = formatAge(time.Now().Add(-end.Sub(j.Status.StartTime.Time)))
		}

		items = append(items, JobInfo{
			Name:        j.Name,
			Namespace:   j.Namespace,
			Completions: fmt.Sprintf("%d/%s", j.Status.Succeeded, completions),
			Succeeded:   j.Status.Succeeded,
			Failed:      j.Status.Failed,
			Active:      j.Status.Active,
			Duration:    duration,
			Age:         formatAge(j.CreationTimestamp.Time),
			Status:      status,
			Conditions:  conds,
		})
	}

	return items, nil
}

// computeNextExecution вычисляет следующее время выполнения cron-расписания.
// Если указана timeZone, используем её, иначе UTC.
func computeNextExecution(schedule string, timeZone *string) string {
	loc := time.UTC
	if timeZone != nil && *timeZone != "" {
		var err error
		loc, err = time.LoadLocation(*timeZone)
		if err != nil {
			loc = time.UTC
		}
	}

	parser := cron.NewParser(cron.Minute | cron.Hour | cron.Dom | cron.Month | cron.Dow)
	spec, err := parser.Parse(schedule)
	if err != nil {
		return "—"
	}

	next := spec.Next(time.Now().In(loc))
	duration := time.Until(next)

	if duration < time.Minute {
		secs := int(duration.Seconds())
		return fmt.Sprintf("In %d seconds", secs)
	}
	if duration < time.Hour {
		mins := int(duration.Minutes())
		if mins == 1 {
			return "In 1 minute"
		}
		return fmt.Sprintf("In %d minutes", mins)
	}
	if duration < 24*time.Hour {
		hours := int(duration.Hours())
		mins := int(duration.Minutes()) % 60
		if mins == 0 {
			if hours == 1 {
				return "In 1 hour"
			}
			return fmt.Sprintf("In %d hours", hours)
		}
		return fmt.Sprintf("In %d hours %d minutes", hours, mins)
	}
	days := int(duration.Hours()) / 24
	if days == 1 {
		return "In 1 day"
	}
	return fmt.Sprintf("In %d days", days)
}

func (m *Manager) ScaleStatefulSet(namespace, name string, replicas int32) error {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	scale, err := conn.Client.AppsV1().StatefulSets(namespace).GetScale(ctx, name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("не удалось получить scale: %w", err)
	}

	scale.Spec.Replicas = replicas

	_, err = conn.Client.AppsV1().StatefulSets(namespace).UpdateScale(ctx, name, scale, metav1.UpdateOptions{})
	if err != nil {
		return fmt.Errorf("не удалось масштабировать: %w", err)
	}

	return nil
}
