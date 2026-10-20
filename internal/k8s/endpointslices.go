package k8s

import (
	"context"
	"fmt"
	"strings"
	"time"

	discoveryv1 "k8s.io/api/discovery/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type SliceEndpointInfo struct {
	Address     string   `json:"address"`
	Ready       bool     `json:"ready"`
	Serving     bool     `json:"serving"`
	Terminating bool     `json:"terminating"`
	Node        string   `json:"node"`
	Zone        string   `json:"zone"`
	Hostname    string   `json:"hostname"`
	Target      *RefInfo `json:"target"`
}

type EndpointSliceInfo struct {
	Name        string              `json:"name"`
	Namespace   string              `json:"namespace"`
	Addresses   []string            `json:"addresses"`
	Ports       []string            `json:"ports"`
	Zone        string              `json:"zone"`
	Endpoints   string              `json:"endpoints"` // ready/total
	AddressType string              `json:"addressType"`
	Service     string              `json:"service"`
	Age         string              `json:"age"`
	Created     string              `json:"created"`
	Labels      map[string]string   `json:"labels"`
	Annotations map[string]string   `json:"annotations"`
	Entries     []SliceEndpointInfo `json:"entries"`
}

// По спецификации nil в ready/serving означает true.
func condTrue(b *bool) bool { return b == nil || *b }

func convertEndpointSlice(es discoveryv1.EndpointSlice) EndpointSliceInfo {
	info := EndpointSliceInfo{
		Name:        es.Name,
		Namespace:   es.Namespace,
		AddressType: string(es.AddressType),
		Service:     es.Labels[discoveryv1.LabelServiceName],
		Age:         formatAge(es.CreationTimestamp.Time),
		Created:     es.CreationTimestamp.Format(time.RFC3339),
		Labels:      es.Labels,
		Annotations: es.Annotations,
		Addresses:   []string{},
		Ports:       []string{},
		Entries:     []SliceEndpointInfo{},
	}
	if info.AddressType == "" {
		info.AddressType = "IPv4"
	}

	ready := 0
	for _, ep := range es.Endpoints {
		e := SliceEndpointInfo{
			Address:     strings.Join(ep.Addresses, ", "),
			Ready:       condTrue(ep.Conditions.Ready),
			Serving:     condTrue(ep.Conditions.Serving),
			Terminating: ep.Conditions.Terminating != nil && *ep.Conditions.Terminating,
		}
		if ep.NodeName != nil {
			e.Node = *ep.NodeName
		}
		if ep.Zone != nil {
			e.Zone = *ep.Zone
			if info.Zone == "" {
				info.Zone = e.Zone
			}
		}
		if ep.Hostname != nil {
			e.Hostname = *ep.Hostname
		}
		if r := ep.TargetRef; r != nil {
			e.Target = &RefInfo{Kind: r.Kind, Name: r.Name, Namespace: r.Namespace}
		}
		if e.Ready {
			ready++
		}
		info.Addresses = append(info.Addresses, ep.Addresses...)
		info.Entries = append(info.Entries, e)
	}
	info.Endpoints = fmt.Sprintf("%d/%d", ready, len(es.Endpoints))

	for _, p := range es.Ports {
		s := ""
		if p.Name != nil && *p.Name != "" {
			s = *p.Name + ":"
		}
		if p.Port != nil {
			s += fmt.Sprintf("%d", *p.Port)
		}
		if p.Protocol != nil {
			s += "/" + string(*p.Protocol)
		}
		if s != "" {
			info.Ports = append(info.Ports, s)
		}
	}
	return info
}

func (m *Manager) GetEndpointSlices(namespace string) ([]EndpointSliceInfo, error) {
	conn, err := m.GetActiveConnection()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	list, err := conn.Client.DiscoveryV1().EndpointSlices(namespace).List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	items := make([]EndpointSliceInfo, 0, len(list.Items))
	for _, es := range list.Items {
		items = append(items, convertEndpointSlice(es))
	}
	return items, nil
}
