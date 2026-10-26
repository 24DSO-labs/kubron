// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/discovery/fake"
	clienttesting "k8s.io/client-go/testing"
)

func TestArgoInstalled(t *testing.T) {
	cases := []struct {
		name      string
		resources []metav1.APIResource
		want      bool
	}{
		{"нет группы", nil, false},
		{"только appprojects", []metav1.APIResource{{Name: "appprojects"}}, false},
		{"есть applications", []metav1.APIResource{{Name: "applications"}, {Name: "appprojects"}}, true},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			fd := &fake.FakeDiscovery{Fake: &clienttesting.Fake{}}
			if tc.resources != nil {
				fd.Resources = []*metav1.APIResourceList{{GroupVersion: argoGroupVersion, APIResources: tc.resources}}
			}
			got, err := argoInstalled(fd)
			if err != nil {
				t.Fatalf("неожиданная ошибка: %v", err)
			}
			if got != tc.want {
				t.Fatalf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestArgoKindsRegistered(t *testing.T) {
	for _, k := range []string{"application", "applicationset", "appproject"} {
		info, ok := kindTable[k]
		if !ok || info.gvr.Group != argoGroup || !info.namespaced {
			t.Errorf("%s не зарегистрирован корректно: %+v", k, info)
		}
	}
}
