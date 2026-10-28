// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package pgstore

import (
	"strings"
	"testing"
)

func TestEmbeddedMigrationsAreValid(t *testing.T) {
	ms, err := Migrations()
	if err != nil {
		t.Fatal(err)
	}
	if len(ms) == 0 {
		t.Fatal("нет миграций")
	}
	for _, m := range ms {
		if strings.TrimSpace(m.Up) == "" || strings.TrimSpace(m.Down) == "" {
			t.Errorf("%04d_%s: пустой up или down", m.Version, m.Name)
		}
	}
}
