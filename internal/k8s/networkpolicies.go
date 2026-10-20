package k8s

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NetworkPolicyRuleInfo — одно правило. Пустой Peers / Ports означает «все».
type NetworkPolicyRuleInfo struct {
	Peers []string `json:"peers"`
	Ports []string `json:"ports"`
}

type NetworkPolicyInfo struct {
	Name        string                  `json:"name"`
	Namespace   string                  `json:"namespace"`
	PodSelector string                  `json:"podSelector"`
	PolicyTypes []string                `json:"policyTypes"`
	Ingress     []NetworkPolicyRuleInfo `json:"ingress"`
	Egress      []NetworkPolicyRuleInfo `json:"egress"`
	Age         string                  `json:"age"`
	Created     string                  `json:"created"`
	Labels      map[string]string       `json:"labels"`
	Annotations map[string]string       `json:"annotations"`
}

// formatSelector: "a=1, b in (x, y)"; пустой селектор даёт "".
func formatSelector(sel *metav1.LabelSelector) string {
	if sel == nil {
		return ""
	}
	var parts []string
	if s := formatMatchLabels(sel); s != "" {
		parts = append(parts, s)
	}
	for _, e := range sel.MatchExpressions {
		op := strings.ToLower(string(e.Operator))
		if len(e.Values) == 0 {
			parts = append(parts, e.Key+" "+op)
		} else {
			parts = append(parts, fmt.Sprintf("%s %s (%s)", e.Key, op, strings.Join(e.Values, ", ")))
		}
	}
	return strings.Join(parts, ", ")
}

func policyPorts(ports []networkingv1.NetworkPolicyPort) []string {
	out := []string{}
	for _, p := range ports {
		s := "TCP"
		if p.Protocol != nil {
			s = string(*p.Protocol)
		}
		if p.Port != nil {
			s += "/" + p.Port.String()
			if p.EndPort != nil {
				s += "-" + strconv.Itoa(int(*p.EndPort))
			}
		}
		out = append(out, s)
	}
	return out
}

func policyPeers(peers []networkingv1.NetworkPolicyPeer) []string {
	out := []string{}
	for _, p := range peers {
		if ib := p.IPBlock; ib != nil {
			s := "ip: " + ib.CIDR
			if len(ib.Except) > 0 {
				s += " except " + strings.Join(ib.Except, ", ")
			}
			out = append(out, s)
			continue
		}
		var parts []string
		if p.NamespaceSelector != nil {
			parts = append(parts, "ns: "+orAll(formatSelector(p.NamespaceSelector)))
		}
		if p.PodSelector != nil {
			parts = append(parts, "pod: "+orAll(formatSelector(p.PodSelector)))
		}
		if len(parts) > 0 {
			out = append(out, strings.Join(parts, " · "))
		}
	}
	return out
}

func orAll(s string) string {
	if s == "" {
		return "all"
	}
	return s
}

func (m *Manager) GetNetworkPolicies(namespace string) ([]NetworkPolicyInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.NetworkingV1().NetworkPolicies(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}

	items := make([]NetworkPolicyInfo, 0, len(list.Items))
	for _, np := range list.Items {
		info := NetworkPolicyInfo{
			Name:        np.Name,
			Namespace:   np.Namespace,
			PodSelector: orAll(formatSelector(&np.Spec.PodSelector)),
			PolicyTypes: policyTypesOf(&np),
			Ingress:     []NetworkPolicyRuleInfo{},
			Egress:      []NetworkPolicyRuleInfo{},
			Age:         formatAge(np.CreationTimestamp.Time),
			Created:     np.CreationTimestamp.Format(time.RFC3339),
			Labels:      np.Labels,
			Annotations: np.Annotations,
		}
		if info.PodSelector == "all" {
			info.PodSelector = "All pods"
		}
		for _, r := range np.Spec.Ingress {
			info.Ingress = append(info.Ingress, NetworkPolicyRuleInfo{Peers: policyPeers(r.From), Ports: policyPorts(r.Ports)})
		}
		for _, r := range np.Spec.Egress {
			info.Egress = append(info.Egress, NetworkPolicyRuleInfo{Peers: policyPeers(r.To), Ports: policyPorts(r.Ports)})
		}
		items = append(items, info)
	}
	return items, nil
}

// policyTypesOf: policyTypes с учётом значения по умолчанию из спецификации.
func policyTypesOf(np *networkingv1.NetworkPolicy) []string {
	out := []string{}
	for _, t := range np.Spec.PolicyTypes {
		out = append(out, string(t))
	}
	if len(out) == 0 {
		out = append(out, "Ingress")
		if len(np.Spec.Egress) > 0 {
			out = append(out, "Egress")
		}
	}
	return out
}
