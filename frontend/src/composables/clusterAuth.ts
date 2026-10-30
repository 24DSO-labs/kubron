// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import type { KubeconfigAuth } from '../types'

/**
 * Что нужно сделать с kubeconfig, прежде чем добавлять кластер.
 * none — ничего; credentials — спросить логин и пароль (oidc-login с паролем: токен получит сервер);
 * authcode и exec — в вебе не подойдёт, объяснить.
 */
export type AuthRequirement = 'none' | 'credentials' | 'authcode' | 'exec'

/**
 * Требование к kubeconfig по способу получения учётных данных. В десктопе (webMode = false) kubectl-плагины
 * работают как обычно и ничего не требуется.
 */
export function authRequirement(auth: KubeconfigAuth | undefined, webMode: boolean): AuthRequirement {
  if (!webMode) return 'none'
  switch (auth) {
    case 'oidc-password': return 'credentials'
    case 'oidc-authcode': return 'authcode'
    case 'exec': return 'exec'
    default: return 'none'
  }
}

/** Можно ли отправлять форму: для oidc-login нужны логин и пароль, для невозможных вариантов отправка закрыта. */
export function authReady(req: AuthRequirement, username: string, password: string): boolean {
  switch (req) {
    case 'none': return true
    case 'credentials': return username.trim().length > 0 && password.length > 0
    default: return false
  }
}
