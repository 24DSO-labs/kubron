// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

//go:build !windows

package wailsplatform

// HideChildConsole нужен только на Windows.
func HideChildConsole() {}
