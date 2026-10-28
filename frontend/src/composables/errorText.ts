// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { translateBackendError } from '../i18n/backendErrors'

/**
 * Текст ошибки из чего угодно (Wails отвечает строкой, остальной код — Error), переведённый на
 * язык интерфейса. Любая ошибка бэкенда показывается пользователю только через эту функцию.
 */
export function errorText(e: unknown): string {
  const raw = e instanceof Error ? e.message : typeof e === 'string' ? e : String(e)
  return translateBackendError(raw)
}
