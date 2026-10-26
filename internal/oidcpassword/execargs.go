// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

package oidcpassword

import (
	"fmt"
	"strings"
)

// ExecParams — то, что можно вынуть из блока exec kubeconfig'а с `kubectl oidc-login get-token`.
type ExecParams struct {
	IssuerURL    string
	ClientID     string
	ClientSecret string
	Scopes       []string
}

// ParseOIDCLoginExec распознаёт exec-блок (command + args) плагина kubelogin и возвращает параметры.
// ok=false — это не oidc-login. Ошибка — oidc-login, но с неподдерживаемым grant'ом.
func ParseOIDCLoginExec(command string, args []string) (p ExecParams, ok bool, err error) {
	isLogin := strings.Contains(command, "oidc-login") || strings.Contains(command, "kubelogin")
	for _, a := range args {
		if a == "oidc-login" {
			isLogin = true
		}
	}
	if !isLogin {
		return p, false, nil
	}
	grant := "authcode" // значение по умолчанию у kubelogin
	for i := 0; i < len(args); i++ {
		name, val, hasVal := strings.Cut(args[i], "=")
		if !strings.HasPrefix(name, "--") {
			continue
		}
		if !hasVal && i+1 < len(args) && !strings.HasPrefix(args[i+1], "--") {
			val = args[i+1]
			i++
		}
		switch name {
		case "--oidc-issuer-url":
			p.IssuerURL = val
		case "--oidc-client-id":
			p.ClientID = val
		case "--oidc-client-secret":
			p.ClientSecret = val
		case "--oidc-extra-scope":
			p.Scopes = append(p.Scopes, val)
		case "--grant-type":
			grant = val
		}
	}
	if grant != "password" {
		return p, true, fmt.Errorf("grant-type %q не поддерживается: в веб-режиме доступен только password", grant)
	}
	if p.IssuerURL == "" || p.ClientID == "" {
		return p, true, fmt.Errorf("в exec-блоке нет --oidc-issuer-url или --oidc-client-id")
	}
	return p, true, nil
}
