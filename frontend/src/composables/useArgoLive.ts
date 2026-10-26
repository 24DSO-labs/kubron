// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue'
import { onEvent } from '../platform/events'

/** Событие бэкенда: Application создан, изменён или удалён (informer, internal/k8s/argocd_cache.go) */
const ARGO_CHANGED = 'argocd:changed'

/**
 * Счётчик изменений Argo CD. Определения ресурсов подписываются на него через `refreshOn`,
 * дашборд — через watch: сам список перезагружается существующим механизмом, без новых каналов.
 * Таймер autoRefresh остаётся страховкой: informer может не стартовать (нет прав на list/watch).
 */
export const argoLiveTick = ref(0)

let off: (() => void) | null = null

/** Подписка идемпотентна: повторный вызов (смена кластера) не плодит обработчики */
export function startArgoLive() {
  if (off) return
  off = onEvent(ARGO_CHANGED, () => { argoLiveTick.value++ })
}

export function stopArgoLive() {
  off?.()
  off = null
}
