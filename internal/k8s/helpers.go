// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package k8s

import (
	"fmt"
	"time"
)

func formatAge(created time.Time) string {
	duration := time.Since(created)

	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60

	if days > 0 {
		return fmt.Sprintf("%dd%dh", days, hours)
	}
	if hours > 0 {
		return fmt.Sprintf("%dh%dm", hours, minutes)
	}
	return fmt.Sprintf("%dm", minutes)
}

func formatMemory(bytes int64) string {
	gb := float64(bytes) / (1024 * 1024 * 1024)
	if gb >= 1 {
		return fmt.Sprintf("%.1f Gi", gb)
	}
	mb := float64(bytes) / (1024 * 1024)
	return fmt.Sprintf("%.0f Mi", mb)
}
