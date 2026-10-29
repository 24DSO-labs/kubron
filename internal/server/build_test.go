// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package server

import (
	"strings"
	"testing"
	"testing/fstest"
)

func TestCheckWebBuild(t *testing.T) {
	web := fstest.MapFS{"index.html": {Data: []byte(`<head><meta name="kubron-target" content="web"></head>`)}}
	if err := CheckWebBuild(web); err != nil {
		t.Errorf("web build rejected: %v", err)
	}
	desktop := fstest.MapFS{"index.html": {Data: []byte(`<head><title>Kubron</title></head>`)}}
	err := CheckWebBuild(desktop)
	if err == nil || !strings.Contains(err.Error(), "build:web") {
		t.Errorf("a desktop build must be rejected with a hint: %v", err)
	}
	if err := CheckWebBuild(fstest.MapFS{}); err != nil {
		t.Errorf("an empty dist is not an error (503 will be served): %v", err)
	}
}
