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
		t.Errorf("веб-сборка отвергнута: %v", err)
	}
	desktop := fstest.MapFS{"index.html": {Data: []byte(`<head><title>Kubron</title></head>`)}}
	err := CheckWebBuild(desktop)
	if err == nil || !strings.Contains(err.Error(), "build:web") {
		t.Errorf("десктопная сборка должна отвергаться с подсказкой: %v", err)
	}
	if err := CheckWebBuild(fstest.MapFS{}); err != nil {
		t.Errorf("пустой dist — не ошибка (будет 503): %v", err)
	}
}
