// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

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
	Name         string              `json:"name"`
	Namespace    string              `json:"namespace"`
	Addresses    []string            `json:"addresses"`
	Ports        []string            `json:"ports"`
	Zone         string              `json:"zone"`
	Endpoints    string              `json:"endpoints"` // ready/total
	AddressType  string              `json:"addressType"`
	Service      string              `json:"service"`
	Age          string              `json:"age"`
	Created      string              `json:"created"`
	Labels       map[string]string   `json:"labels"`
	Annotations  map[string]string   `json:"annotations"`
	Entries      []SliceEndpointInfo `json:"entries"`
	EndpointList string              `json:"endpointList"` // "ip:port, ip:port +N"
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

	var portNums []int32
	for _, p := range es.Ports {
		if p.Port != nil {
			portNums = append(portNums, *p.Port)
		}
	}
	var flat []string

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
			for _, a := range ep.Addresses {
				if len(portNums) == 0 {
					flat = append(flat, a)
					continue
				}
				for _, p := range portNums {
					flat = append(flat, hostPort(a, p))
				}
			}
		}
		info.Addresses = append(info.Addresses, ep.Addresses...)
		info.Entries = append(info.Entries, e)
	}
	info.Endpoints = fmt.Sprintf("%d/%d", ready, len(es.Endpoints))
	info.EndpointList = formatEndpointList(flat)

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

func (m *Manager) GetEndpointSlices(namespaces []string) ([]EndpointSliceInfo, error) {
	return scopedList(m, "GetEndpointSlices", "discovery.k8s.io", "endpointslices", namespaces, listEndpointSlices,
		func(it EndpointSliceInfo) string { return it.Namespace })
}

// listEndpointSlices читает ресурсы одного namespace ("" — все) для GetEndpointSlices.
func listEndpointSlices(ctx context.Context, conn *ClusterConnection, namespace string) ([]EndpointSliceInfo, error) {
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
