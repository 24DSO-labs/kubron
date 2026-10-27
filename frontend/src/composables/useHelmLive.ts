// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue'
import { onEvent } from '../platform/events'

/** Событие бэкенда: секрет Helm-релиза изменился (informer, internal/k8s/helm_cache.go) */
const HELM_CHANGED = 'helm:changed'

/**
 * Счётчик изменений Helm-релизов. Определения ресурсов подписываются на него через `refreshOn`,
 * карточка Overview — через watch. Таймер autoRefresh остаётся страховкой: informer не стартует,
 * если у пользователя нет права list на секреты.
 */
export const helmLiveTick = ref(0)

let off: (() => void) | null = null

/** Подписка идемпотентна: повторный вызов не плодит обработчики */
export function startHelmLive() {
  if (off) return
  off = onEvent(HELM_CHANGED, () => { helmLiveTick.value++ })
}

export function stopHelmLive() {
  off?.()
  off = null
}
