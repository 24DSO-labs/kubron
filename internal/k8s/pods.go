package k8s

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type VolumeInfo struct {
	Name    string            `json:"name"`
	Type    string            `json:"type"`
	Details map[string]string `json:"details"`
}

type ConditionInfo struct {
	Type               string `json:"type"`
	Status             string `json:"status"`
	LastTransitionTime string `json:"lastTransitionTime"`
	Reason             string `json:"reason"`
	Message            string `json:"message"`
}

type TolerationInfo struct {
	Key               string `json:"key"`
	Operator          string `json:"operator"`
	Value             string `json:"value"`
	Effect            string `json:"effect"`
	TolerationSeconds *int64 `json:"tolerationSeconds"`
}

type PodInfo struct {
	Name            string                  `json:"name"`
	Namespace       string                  `json:"namespace"`
	Status          string                  `json:"status"`
	Ready           string                  `json:"ready"`
	Restarts        int32                   `json:"restarts"`
	Age             string                  `json:"age"`
	Node            string                  `json:"node"`
	IP              string                  `json:"ip"`
	Labels          map[string]string       `json:"labels"`
	Annotations     map[string]string       `json:"annotations"`
	ControlledBy    string                  `json:"controlledBy"`
	ServiceAccount  string                  `json:"serviceAccount"`
	NodeSelector    map[string]string       `json:"nodeSelector"`
	Conditions      []ConditionInfo         `json:"conditions"`
	Tolerations     []TolerationInfo        `json:"tolerations"`
	PodAntiAffinity *corev1.PodAntiAffinity `json:"podAntiAffinity"`
	Containers      []ContainerInfo         `json:"containers"`
	Volumes         []VolumeInfo            `json:"volumes"`
	QOS             string                  `json:"qos"`
	Created         string                  `json:"created"`
}

type ContainerInfo struct {
	Name         string `json:"name"`
	Image        string `json:"image"`
	Ready        bool   `json:"ready"`
	RestartCount int32  `json:"restartCount"`
	State        string `json:"state"`
	IsInit       bool   `json:"isInit"`
}

func (m *Manager) GetPods(namespace string) ([]PodInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	podList, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var pods []PodInfo
	for _, pod := range podList.Items {
		pods = append(pods, convertPod(pod))
	}

	return pods, nil
}

func convertPod(pod corev1.Pod) PodInfo {
	var totalRestarts int32
	var readyCount int
	var containers []ContainerInfo
	var conditions []ConditionInfo
	var tolerations []TolerationInfo

	// Process init containers
	for _, ics := range pod.Status.InitContainerStatuses {
		state := "unknown"
		if ics.State.Running != nil {
			state = "running"
		} else if ics.State.Waiting != nil {
			state = ics.State.Waiting.Reason
		} else if ics.State.Terminated != nil {
			state = ics.State.Terminated.Reason
		}

		containers = append(containers, ContainerInfo{
			Name:         ics.Name,
			Image:        ics.Image,
			Ready:        ics.Ready,
			RestartCount: ics.RestartCount,
			State:        state,
			IsInit:       true,
		})
	}

	// Process regular containers
	for _, cs := range pod.Status.ContainerStatuses {
		totalRestarts += cs.RestartCount
		if cs.Ready {
			readyCount++
		}

		state := "unknown"
		if cs.State.Running != nil {
			state = "running"
		} else if cs.State.Waiting != nil {
			state = cs.State.Waiting.Reason
		} else if cs.State.Terminated != nil {
			state = cs.State.Terminated.Reason
		}

		containers = append(containers, ContainerInfo{
			Name:         cs.Name,
			Image:        cs.Image,
			Ready:        cs.Ready,
			RestartCount: cs.RestartCount,
			State:        state,
			IsInit:       false,
		})
	}

	// Conditions
	for _, cond := range pod.Status.Conditions {
		conditions = append(conditions, ConditionInfo{
			Type:               string(cond.Type),
			Status:             string(cond.Status),
			LastTransitionTime: cond.LastTransitionTime.Format(time.RFC3339),
			Reason:             cond.Reason,
			Message:            cond.Message,
		})
	}

	// Tolerations
	for _, tol := range pod.Spec.Tolerations {
		tolerations = append(tolerations, TolerationInfo{
			Key:               tol.Key,
			Operator:          string(tol.Operator),
			Value:             tol.Value,
			Effect:            string(tol.Effect),
			TolerationSeconds: tol.TolerationSeconds,
		})
	}

	controlledBy := ""
	if len(pod.OwnerReferences) > 0 {
		ref := pod.OwnerReferences[0]
		controlledBy = fmt.Sprintf("%s/%s", ref.Kind, ref.Name)
	}

	var podAntiAffinity *corev1.PodAntiAffinity
	if pod.Spec.Affinity != nil {
		podAntiAffinity = pod.Spec.Affinity.PodAntiAffinity
	}

	qos := string(pod.Status.QOSClass)
	created := pod.CreationTimestamp.Format(time.RFC3339)

	status := string(pod.Status.Phase)
	// Более точный статус
	for _, cs := range pod.Status.ContainerStatuses {
		if cs.State.Waiting != nil && cs.State.Waiting.Reason != "" {
			status = cs.State.Waiting.Reason
			break
		}
		if cs.State.Terminated != nil && cs.State.Terminated.Reason != "" {
			status = cs.State.Terminated.Reason
			break
		}
	}

	var volumes []VolumeInfo
	for _, vol := range pod.Spec.Volumes {
		v := VolumeInfo{
			Name:    vol.Name,
			Details: make(map[string]string),
		}
		switch {
		case vol.HostPath != nil:
			v.Type = "HostPath"
			v.Details["Path"] = vol.HostPath.Path
			if vol.HostPath.Type != nil {
				v.Details["Type"] = string(*vol.HostPath.Type)
			}

		case vol.EmptyDir != nil:
			v.Type = "EmptyDir"
			if vol.EmptyDir.Medium != "" {
				v.Details["Medium"] = string(vol.EmptyDir.Medium)
			}
			if vol.EmptyDir.SizeLimit != nil {
				v.Details["SizeLimit"] = vol.EmptyDir.SizeLimit.String()
			}

		case vol.Secret != nil:
			v.Type = "Secret"
			v.Details["SecretName"] = vol.Secret.SecretName
			v.Details["Items"] = fmt.Sprintf("%d", len(vol.Secret.Items))
			if vol.Secret.DefaultMode != nil {
				v.Details["DefaultFileMode"] = fmt.Sprintf("0%o", *vol.Secret.DefaultMode)
			}
			if vol.Secret.Optional != nil {
				v.Details["Optional"] = fmt.Sprintf("%t", *vol.Secret.Optional)
			}

		case vol.ConfigMap != nil:
			v.Type = "ConfigMap"
			v.Details["ConfigMapName"] = vol.ConfigMap.Name
			v.Details["Items"] = fmt.Sprintf("%d", len(vol.ConfigMap.Items))
			if vol.ConfigMap.DefaultMode != nil {
				v.Details["DefaultFileMode"] = fmt.Sprintf("0%o", *vol.ConfigMap.DefaultMode)
			}
			if vol.ConfigMap.Optional != nil {
				v.Details["Optional"] = fmt.Sprintf("%t", *vol.ConfigMap.Optional)
			}

		case vol.Projected != nil:
			v.Type = "Projected"
			v.Details["Sources"] = fmt.Sprintf("%d", len(vol.Projected.Sources))
			if vol.Projected.DefaultMode != nil {
				v.Details["DefaultMode"] = fmt.Sprintf("0%o", *vol.Projected.DefaultMode)
			}

		case vol.PersistentVolumeClaim != nil:
			v.Type = "PersistentVolumeClaim"
			v.Details["ClaimName"] = vol.PersistentVolumeClaim.ClaimName
			v.Details["ReadOnly"] = fmt.Sprintf("%t", vol.PersistentVolumeClaim.ReadOnly)

		case vol.NFS != nil:
			v.Type = "NFS"
			v.Details["Server"] = vol.NFS.Server
			v.Details["Path"] = vol.NFS.Path
			v.Details["ReadOnly"] = fmt.Sprintf("%t", vol.NFS.ReadOnly)

		case vol.CSI != nil:
			v.Type = "CSI"
			v.Details["Driver"] = vol.CSI.Driver
			if vol.CSI.ReadOnly != nil {
				v.Details["ReadOnly"] = fmt.Sprintf("%t", *vol.CSI.ReadOnly)
			}

		case vol.DownwardAPI != nil:
			v.Type = "DownwardAPI"
			v.Details["Items"] = fmt.Sprintf("%d", len(vol.DownwardAPI.Items))
			if vol.DownwardAPI.DefaultMode != nil {
				v.Details["DefaultMode"] = fmt.Sprintf("0%o", *vol.DownwardAPI.DefaultMode)
			}

		default:
			v.Type = "Other"
		}
		volumes = append(volumes, v)
	}

	return PodInfo{
		Name:            pod.Name,
		Namespace:       pod.Namespace,
		Status:          status,
		Ready:           fmt.Sprintf("%d/%d", readyCount, len(pod.Spec.Containers)),
		Restarts:        totalRestarts,
		Age:             formatAge(pod.CreationTimestamp.Time),
		Node:            pod.Spec.NodeName,
		IP:              pod.Status.PodIP,
		Labels:          pod.Labels,
		Annotations:     pod.Annotations,
		ControlledBy:    controlledBy,
		ServiceAccount:  pod.Spec.ServiceAccountName,
		NodeSelector:    pod.Spec.NodeSelector,
		Conditions:      conditions,
		Tolerations:     tolerations,
		PodAntiAffinity: podAntiAffinity,
		Containers:      containers,
		Volumes:         volumes,
		QOS:             qos,
		Created:         created,
	}
}
