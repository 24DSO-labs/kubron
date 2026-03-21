package k8s

import (
	"context"
	"fmt"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PVInfo struct {
	Name          string `json:"name"`
	Capacity      string `json:"capacity"`
	AccessModes   string `json:"accessModes"`
	ReclaimPolicy string `json:"reclaimPolicy"`
	Status        string `json:"status"`
	Claim         string `json:"claim"`
	StorageClass  string `json:"storageClass"`
	Age           string `json:"age"`
}

type PVCInfo struct {
	Name         string `json:"name"`
	Namespace    string `json:"namespace"`
	Status       string `json:"status"`
	Volume       string `json:"volume"`
	Capacity     string `json:"capacity"`
	AccessModes  string `json:"accessModes"`
	StorageClass string `json:"storageClass"`
	Age          string `json:"age"`
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

	var items []PVInfo
	for _, pv := range list.Items {
		capacity := ""
		if storage, ok := pv.Spec.Capacity["storage"]; ok {
			capacity = storage.String()
		}

		var modes []string
		for _, m := range pv.Spec.AccessModes {
			modes = append(modes, string(m))
		}

		claim := "—"
		if pv.Spec.ClaimRef != nil {
			claim = fmt.Sprintf("%s/%s", pv.Spec.ClaimRef.Namespace, pv.Spec.ClaimRef.Name)
		}

		sc := ""
		if pv.Spec.StorageClassName != "" {
			sc = pv.Spec.StorageClassName
		}

		items = append(items, PVInfo{
			Name:          pv.Name,
			Capacity:      capacity,
			AccessModes:   joinShort(modes),
			ReclaimPolicy: string(pv.Spec.PersistentVolumeReclaimPolicy),
			Status:        string(pv.Status.Phase),
			Claim:         claim,
			StorageClass:  sc,
			Age:           formatAge(pv.CreationTimestamp.Time),
		})
	}
	return items, nil
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

	var items []PVCInfo
	for _, pvc := range list.Items {
		capacity := ""
		if pvc.Status.Capacity != nil {
			if storage, ok := pvc.Status.Capacity["storage"]; ok {
				capacity = storage.String()
			}
		}

		var modes []string
		for _, m := range pvc.Spec.AccessModes {
			modes = append(modes, string(m))
		}

		sc := ""
		if pvc.Spec.StorageClassName != nil {
			sc = *pvc.Spec.StorageClassName
		}

		items = append(items, PVCInfo{
			Name:         pvc.Name,
			Namespace:    pvc.Namespace,
			Status:       string(pvc.Status.Phase),
			Volume:       pvc.Spec.VolumeName,
			Capacity:     capacity,
			AccessModes:  joinShort(modes),
			StorageClass: sc,
			Age:          formatAge(pvc.CreationTimestamp.Time),
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
