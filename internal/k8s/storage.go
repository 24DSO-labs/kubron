package k8s

import (
	"context"
	"fmt"
	"sort"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVInfo struct {
	Name           string `json:"name"`
	Capacity       string `json:"capacity"`
	AccessModes    string `json:"accessModes"`
	ReclaimPolicy  string `json:"reclaimPolicy"`
	Status         string `json:"status"`
	Claim          string `json:"claim"` // "ns/name" или "—"
	ClaimNamespace string `json:"claimNamespace"`
	ClaimName      string `json:"claimName"`
	StorageClass   string `json:"storageClass"`
	VolumeMode     string `json:"volumeMode"`

	Age         string            `json:"age"`
	Created     string            `json:"created"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
	Finalizers  []string          `json:"finalizers"`
	Provider    *PVProviderInfo   `json:"provider"`
}

// PVField — строка вкладки Provider (имя, значение), порядок сохраняется.
type PVField struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// PVProviderInfo — откуда берётся том и его параметры.
type PVProviderInfo struct {
	Source string    `json:"source"` // Container Storage Interface, NFS, HostPath ...
	Fields []PVField `json:"fields"`
}

type PVCSelectorInfo struct {
	MatchLabels      map[string]string `json:"matchLabels"`
	MatchExpressions []MatchExpression `json:"matchExpressions"`
}
type PVCInfo struct {
	Name         string            `json:"name"`
	Namespace    string            `json:"namespace"`
	Status       string            `json:"status"`
	Volume       string            `json:"volume"`
	Capacity     string            `json:"capacity"`  // status.capacity, а пока не bound — requested
	Requested    string            `json:"requested"` // spec.resources.requests.storage
	AccessModes  string            `json:"accessModes"`
	StorageClass string            `json:"storageClass"`
	VolumeMode   string            `json:"volumeMode"`
	Age          string            `json:"age"`
	Created      string            `json:"created"`
	Labels       map[string]string `json:"labels"`
	Annotations  map[string]string `json:"annotations"`
	Finalizers   []string          `json:"finalizers"`
	Pods         []string          `json:"pods"`
	Selector     *PVCSelectorInfo  `json:"selector"`
}

func (m *Manager) GetPersistentVolumes() ([]PVInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().PersistentVolumes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]PVInfo, 0, len(list.Items))
	for _, pv := range list.Items {
		capacity := ""
		if q, ok := pv.Spec.Capacity[corev1.ResourceStorage]; ok {
			capacity = q.String()
		}

		var modes []string
		for _, am := range pv.Spec.AccessModes {
			modes = append(modes, string(am))
		}

		claim, claimNS, claimName := "—", "", ""
		if ref := pv.Spec.ClaimRef; ref != nil {
			claimNS, claimName = ref.Namespace, ref.Name
			claim = ref.Namespace + "/" + ref.Name
		}

		volumeMode := ""
		if pv.Spec.VolumeMode != nil {
			volumeMode = string(*pv.Spec.VolumeMode)
		}

		items = append(items, PVInfo{
			Name:           pv.Name,
			Capacity:       capacity,
			AccessModes:    joinShort(modes),
			ReclaimPolicy:  string(pv.Spec.PersistentVolumeReclaimPolicy),
			Status:         string(pv.Status.Phase),
			Claim:          claim,
			ClaimNamespace: claimNS,
			ClaimName:      claimName,
			StorageClass:   pv.Spec.StorageClassName,
			VolumeMode:     volumeMode,
			Provider:       pvProvider(pv.Spec.PersistentVolumeSource),
			Age:            formatAge(pv.CreationTimestamp.Time),
			Created:        pv.CreationTimestamp.Format(time.RFC3339),
			Labels:         pv.Labels,
			Annotations:    pv.Annotations,
			Finalizers:     pv.Finalizers,
		})
	}
	return items, nil
}

func pvProvider(s corev1.PersistentVolumeSource) *PVProviderInfo {
	var p *PVProviderInfo
	add := func(name, value string) {
		if value != "" {
			p.Fields = append(p.Fields, PVField{Name: name, Value: value})
		}
	}

	switch {
	case s.CSI != nil:
		p = &PVProviderInfo{Source: "Container Storage Interface"}
		add("Driver", s.CSI.Driver)
		keys := make([]string, 0, len(s.CSI.VolumeAttributes))
		for k := range s.CSI.VolumeAttributes {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, k := range keys {
			add(k, s.CSI.VolumeAttributes[k])
		}
		add("FS Type", s.CSI.FSType)
		add("Read Only", fmt.Sprintf("%t", s.CSI.ReadOnly))
	case s.NFS != nil:
		p = &PVProviderInfo{Source: "NFS"}
		add("Server", s.NFS.Server)
		add("Path", s.NFS.Path)
		add("Read Only", fmt.Sprintf("%t", s.NFS.ReadOnly))
	case s.HostPath != nil:
		p = &PVProviderInfo{Source: "HostPath"}
		add("Path", s.HostPath.Path)
		if s.HostPath.Type != nil {
			add("Type", string(*s.HostPath.Type))
		}
	case s.Local != nil:
		p = &PVProviderInfo{Source: "Local"}
		add("Path", s.Local.Path)
		if s.Local.FSType != nil {
			add("FS Type", *s.Local.FSType)
		}
	case s.ISCSI != nil:
		p = &PVProviderInfo{Source: "iSCSI"}
		add("Target Portal", s.ISCSI.TargetPortal)
		add("IQN", s.ISCSI.IQN)
		add("LUN", fmt.Sprintf("%d", s.ISCSI.Lun))
		add("FS Type", s.ISCSI.FSType)
		add("Read Only", fmt.Sprintf("%t", s.ISCSI.ReadOnly))
	}
	return p
}

// pvcPods: "ns/claim" -> имена активных подов, монтирующих claim. Ошибка не ломает список.
func pvcPods(ctx context.Context, conn *ClusterConnection, namespace string) map[string][]string {
	out := map[string][]string{}
	pods, err := conn.Client.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return out
	}
	for i := range pods.Items {
		p := &pods.Items[i]
		if p.Status.Phase == corev1.PodSucceeded || p.Status.Phase == corev1.PodFailed {
			continue
		}
		seen := map[string]bool{}
		for _, v := range p.Spec.Volumes {
			if v.PersistentVolumeClaim == nil {
				continue
			}
			key := p.Namespace + "/" + v.PersistentVolumeClaim.ClaimName
			if seen[key] {
				continue
			}
			seen[key] = true
			out[key] = append(out[key], p.Name)
		}
	}
	for k := range out {
		sort.Strings(out[k])
	}
	return out
}

func (m *Manager) GetPersistentVolumeClaims(namespace string) ([]PVCInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.CoreV1().PersistentVolumeClaims(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	podsByClaim := pvcPods(ctx, conn, namespace)

	items := make([]PVCInfo, 0, len(list.Items))
	for _, pvc := range list.Items {
		requested := ""
		if q, ok := pvc.Spec.Resources.Requests[corev1.ResourceStorage]; ok {
			requested = q.String()
		}
		capacity := requested
		if q, ok := pvc.Status.Capacity[corev1.ResourceStorage]; ok {
			capacity = q.String()
		}

		var modes []string
		for _, am := range pvc.Spec.AccessModes {
			modes = append(modes, string(am))
		}

		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}
		volumeMode := ""
		if pvc.Spec.VolumeMode != nil {
			volumeMode = string(*pvc.Spec.VolumeMode)
		}

		var selector *PVCSelectorInfo
		if s := pvc.Spec.Selector; s != nil {
			selector = &PVCSelectorInfo{MatchLabels: s.MatchLabels, MatchExpressions: []MatchExpression{}}
			for _, e := range s.MatchExpressions {
				selector.MatchExpressions = append(selector.MatchExpressions, MatchExpression{
					Key: e.Key, Operator: string(e.Operator), Values: e.Values,
				})
			}
		}

		pods := podsByClaim[pvc.Namespace+"/"+pvc.Name]
		if pods == nil {
			pods = []string{}
		}

		items = append(items, PVCInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Volume:       pvc.Spec.VolumeName,
			Capacity:     capacity,
			Requested:    requested,
			AccessModes:  joinShort(modes),
			StorageClass: sc,
			VolumeMode:   volumeMode,
			Age:          formatAge(pvc.CreationTimestamp.Time),
			Created:      pvc.CreationTimestamp.Format(time.RFC3339),
			Labels:       pvc.Labels,
			Annotations:  pvc.Annotations,
			Finalizers:   pvc.Finalizers,
			Pods:         pods,
			Selector:     selector,
		})
	}
	return items, nil
}

type StorageClassInfo struct {
	Name          string `json:"name"`
	Provisioner   string `json:"provisioner"`
	ReclaimPolicy string `json:"reclaimPolicy"`
	Default       bool   `json:"default"`
	Age           string `json:"age"`
}

func (m *Manager) GetStorageClasses() ([]StorageClassInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.StorageV1().StorageClasses().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	var items []StorageClassInfo
	for _, sc := range list.Items {
		isDefault := false
		if val, ok := sc.Annotations["storageclass.kubernetes.io/is-default-class"]; ok {
			isDefault = val == "true"
		}

		provisioner := ""
		if sc.Provisioner != "" {
			provisioner = sc.Provisioner
		}

		reclaimPolicy := ""
		if sc.ReclaimPolicy != nil {
			reclaimPolicy = string(*sc.ReclaimPolicy)
		}

		items = append(items, StorageClassInfo{
			Name:          sc.Name,
			Provisioner:   provisioner,
			ReclaimPolicy: reclaimPolicy,
			Default:       isDefault,
			Age:           formatAge(sc.CreationTimestamp.Time),
		})
	}
	return items, nil
}

func joinShort(modes []string) string {
	result := ""
	for i, m := range modes {
		if i > 0 {
			result += ","
		}
		switch m {
		case "ReadWriteOnce":
			result += "RWO"
		case "ReadOnlyMany":
			result += "ROX"
		case "ReadWriteMany":
			result += "RWX"
		default:
			result += m
		}
	}
	return result
}
