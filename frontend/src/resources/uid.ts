// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Уникальный id для вкладок и сессий. crypto.randomUUID доступен только в «безопасном
 * контексте» (в некоторых WebView для кастомной схемы его может не быть) — тогда запасной вариант.
 */
export function uid(): string {
  if (typeof crypto !== 'undefined' && typeof crypto.randomUUID === 'function') return crypto.randomUUID()
  return `${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}
