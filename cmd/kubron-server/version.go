// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package main

// version — версия сборки. Образ и релиз задают её через -ldflags "-X main.version=<версия>";
// без этого (go run, go build) остаётся "dev".
var version = "dev"
