// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package store

import (
	"github.com/VadimIl224/kubron/internal/audit"
	"github.com/VadimIl224/kubron/internal/auth"
	"github.com/VadimIl224/kubron/internal/k8s"
	"github.com/VadimIl224/kubron/pkg/ext"
)

// FromExt адаптирует публичное хранилище (pkg/ext), например серверное на PostgreSQL из другого
// модуля, к внутреннему Store.
func FromExt(s ext.Store) Store { return extStore{s} }

type extStore struct{ s ext.Store }

func (e extStore) Users() auth.Backend        { return auth.FromExt(e.s.Users()) }
func (e extStore) Audit() audit.Backend       { return e.s.Audit() }
func (e extStore) ArgoTokens() k8s.TokenStore { return e.s.ArgoTokens() }
