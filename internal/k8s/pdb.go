package k8s

import (
	"context"
	"time"

	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type PDBInfo struct {
	Name               string `json:"name"`
	Namespace          string `json:"namespace"`
	MinAvailable       string `json:"minAvailable"`
	MaxUnavailable     string `json:"maxUnavailable"`
	DisruptionsAllowed int32  `json:"disruptionsAllowed"`
	CurrentHealthy     int32  `json:"currentHealthy"`
	DesiredHealthy     int32  `json:"desiredHealthy"`
	Age                string `json:"age"`

	ExpectedPods               int32             `json:"expectedPods"`
	UnhealthyPodEvictionPolicy string            `json:"unhealthyPodEvictionPolicy"`
	Selector                   *PVCSelectorInfo  `json:"selector"`
	SelectorText               string            `json:"selectorText"`
	Conditions                 []ConditionInfo   `json:"conditions"`
	Created                    string            `json:"created"`
	Labels                     map[string]string `json:"labels"`
	ControlledBy               string            `json:"controlledBy"`
	ManagedBy                  string            `json:"managedBy"`
	Annotations                map[string]string `json:"annotations"`
	Finalizers                 []string          `json:"finalizers"`
}

func (m *Manager) GetPodDisruptionBudgets(namespace string) ([]PDBInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.PolicyV1().PodDisruptionBudgets(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	pdbs := make([]PDBInfo, 0, len(list.Items))
	for _, pdb := range list.Items {
		pdbs = append(pdbs, convertPDB(pdb))
	}

	return pdbs, nil
}

func convertPDB(pdb policyv1.PodDisruptionBudget) PDBInfo {
	minAvail := "—"
	if pdb.Spec.MinAvailable != nil {
		minAvail = pdb.Spec.MinAvailable.String()
	}

	maxUnav := "—"
	if pdb.Spec.MaxUnavailable != nil {
		maxUnav = pdb.Spec.MaxUnavailable.String()
	}

	policy := ""
	if pdb.Spec.UnhealthyPodEvictionPolicy != nil {
		policy = string(*pdb.Spec.UnhealthyPodEvictionPolicy)
	}

	conditions := make([]ConditionInfo, 0, len(pdb.Status.Conditions))
	for _, c := range pdb.Status.Conditions {
		conditions = append(conditions, ConditionInfo{
			Type:               c.Type,
			Status:             string(c.Status),
			LastTransitionTime: formatTime(c.LastTransitionTime),
			Reason:             c.Reason,
			Message:            c.Message,
		})
	}

	meta := metaOf(pdb.ObjectMeta)
	return PDBInfo{
		Name:                       pdb.Name,
		Namespace:                  pdb.Namespace,
		MinAvailable:               minAvail,
		MaxUnavailable:             maxUnav,
		DisruptionsAllowed:         pdb.Status.DisruptionsAllowed,
		CurrentHealthy:             pdb.Status.CurrentHealthy,
		DesiredHealthy:             pdb.Status.DesiredHealthy,
		Age:                        meta.Age,
		ExpectedPods:               pdb.Status.ExpectedPods,
		UnhealthyPodEvictionPolicy: policy,
		Selector:                   labelSelectorInfo(pdb.Spec.Selector),
		SelectorText:               formatSelectorcm(pdb.Spec.Selector),
		Conditions:                 conditions,
		Created:                    meta.Created,
		Labels:                     meta.Labels,
		ControlledBy:               controlledByOf(pdb.OwnerReferences),
		ManagedBy:                  managedByOf(pdb.ObjectMeta),
		Annotations:                meta.Annotations,
		Finalizers:                 meta.Finalizers,
	}
}
