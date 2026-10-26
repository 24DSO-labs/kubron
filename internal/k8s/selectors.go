// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"sort"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// formatMatchLabels возвращает selector в виде "a=1, b=2" с детерминированным
// (отсортированным по ключу) порядком: обход map даёт случайный порядок, из-за
// чего значение «мигало» при каждом автообновлении.
func formatMatchLabels(sel *metav1.LabelSelector) string {
	if sel == nil || len(sel.MatchLabels) == 0 {
		return ""
	}
	parts := make([]string, 0, len(sel.MatchLabels))
	for k, v := range sel.MatchLabels {
		parts = append(parts, k+"="+v)
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}
