// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { EventsOn } from '../../wailsjs/runtime/runtime'
import { onEvent as onWebEvent } from './events.web'

/** Отписка от события. */
export type Unsubscribe = () => void

/**
 * Подписка на событие бэкенда (cluster:changed, logs:data:<id>, terminal:data:<id> и т. д.).
 * Десктоп: Wails EventsOn. Веб: WebSocket /api/ws (этап 1).
 */
export function onEvent(name: string, handler: (...args: any[]) => void): Unsubscribe {
  if (import.meta.env.VITE_TARGET === 'web') return onWebEvent(name, handler)
  return EventsOn(name, handler)
}
