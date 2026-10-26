// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"testing"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

const gib = int64(1024 * 1024 * 1024)

func TestFormatBytesIEC(t *testing.T) {
	cases := []struct {
		in   int64
		want string
	}{
		{0, "0"},
		{-5, "0"},
		{512, "512B"},
		{2048, "2.0KiB"},
		{8 * gib, "8.0GiB"},
		{7*gib + 600*1024*1024, "7.6GiB"},
	}
	for _, c := range cases {
		if got := formatBytesIEC(c.in); got != c.want {
			t.Errorf("formatBytesIEC(%d) = %q, want %q", c.in, got, c.want)
		}
	}
}

func TestNodeResourceFields(t *testing.T) {
	rl := corev1.ResourceList{}
	rl[corev1.ResourcePods] = resource.MustParse("110")
	rl[corev1.ResourceCPU] = resource.MustParse("4")
	rl[corev1.ResourceMemory] = resource.MustParse("8Gi")
	rl["nvidia.com/gpu"] = resource.MustParse("1")

	got := nodeResourceFields(rl)
	want := []NodeResourceField{
		{"CPU", "4"},
		{"Memory", "8.0GiB"},
		{"Ephemeral Storage", "0"},
		{"Hugepages-1Gi", "0"},
		{"Hugepages-2Mi", "0"},
		{"Pods", "110"},
		{"nvidia.com/gpu", "1"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %d fields, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("field %d = %+v, want %+v", i, got[i], want[i])
		}
	}
}
