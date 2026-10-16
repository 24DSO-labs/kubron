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

	var pdbs []PDBInfo
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

	return PDBInfo{
		Name:               pdb.Name,
		Namespace:          pdb.Namespace,
		MinAvailable:       minAvail,
		MaxUnavailable:     maxUnav,
		DisruptionsAllowed: pdb.Status.DisruptionsAllowed,
		CurrentHealthy:     pdb.Status.CurrentHealthy,
		DesiredHealthy:     pdb.Status.DesiredHealthy,
		Age:                formatAge(pdb.CreationTimestamp.Time),
	}
}

func (m *Manager) DeletePDB(namespace, name string) error {
conn, err := m.GetActiveConnection()
if err != nil {
return err
}

ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
defer cancel()

return conn.Client.PolicyV1().PodDisruptionBudgets(namespace).Delete(ctx, name, metav1.DeleteOptions{})
}
