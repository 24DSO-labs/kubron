// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package wailsplatform

import "testing"

func TestDesktopCapabilities(t *testing.T) {
	c := (&Platform{}).Capabilities()
	if !c.FileDialog || !c.PortForward || !c.NodeShell || !c.LocalKubeconfig {
		t.Fatalf("в десктопе локальные возможности должны быть включены: %+v", c)
	}
	if c.SSO {
		t.Fatal("SSO в десктопе не предлагается")
	}
}
