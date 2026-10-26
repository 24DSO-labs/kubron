// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

// Package frontend встраивает собранный фронтенд (frontend/dist) в Go-бинарник сервера.
// Для Go-проверок без сборки фронтенда достаточно заглушки: mkdir -p frontend/dist && touch frontend/dist/.keep
package frontend

import "embed"

// Dist — содержимое frontend/dist (корень — каталог dist).
//
//go:embed all:dist
var Dist embed.FS
