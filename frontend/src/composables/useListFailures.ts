// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, getCurrentInstance } from 'vue'
import { onEvent } from '../platform/events'
import { useNotifications } from './useNotifications'

function t(key: string, params?: Record<string, unknown>): string {
  const instance = getCurrentInstance()
  if (instance?.appContext?.config?.globalProperties?.$t) {
    return (instance.appContext.config.globalProperties.$t as any)(key, params)
  }
  return key
}

/** Событие бэкенда list:partial: часть namespace набора прочитать не удалось. */
export interface PartialListPayload {
  method: string
  failures: { namespace: string; error: string }[]
}

export const EVENT_LIST_PARTIAL = 'list:partial'

/** namespace -> текст последней ошибки; по нему NamespaceSelect рисует значок «нет доступа» */
const failedNamespaces = ref<Record<string, string>>({})
/** Наборы отказавших namespace, о которых уже показан тост: автообновление не плодит дубли */
const notified = new Set<string>()

/** Забыть отказы: при смене набора или кластера новая попытка снова может показать тост. */
export function resetListFailures() {
  failedNamespaces.value = {}
  notified.clear()
}

/** Обработчик события; вынесен отдельно ради тестов. Возвращает true, если показан тост. */
export function handlePartialList(payload: PartialListPayload | undefined | null, notify: (msg: string) => void): boolean {
  const failures = payload?.failures ?? []
  if (!failures.length) return false

  failedNamespaces.value = {
    ...failedNamespaces.value,
    ...Object.fromEntries(failures.map(f => [f.namespace, f.error])),
  }

  const names = [...new Set(failures.map(f => f.namespace))].sort()
  const key = names.join(',')
  if (notified.has(key)) return false
  notified.add(key)

  const shown = names.slice(0, 3).join(', ') + (names.length > 3 ? ` ${t('helm.helm_card.more', { count: names.length - 3 })}` : '')
  notify(t('helm.list_failures.noAccess', { shown }))
  return true
}

/** Подписка на событие; вызывать один раз при старте приложения. Возвращает функцию отписки. */
export function listenListFailures(): () => void {
  const { notifyError } = useNotifications()
  return onEvent(EVENT_LIST_PARTIAL, (payload: PartialListPayload) => { handlePartialList(payload, notifyError) })
}

export function useListFailures() {
  return { failedNamespaces, resetListFailures }
}
