// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { t, te } from '../i18n'

/** Результат действия; значения совпадают с internal/audit */
export type AuditResult = 'ok' | 'failed' | 'denied'

/** Запись журнала (internal/audit.Event) */
export interface AuditEvent {
  time: string
  user: string
  role?: string
  action: string
  result: AuditResult | string
  detail?: string
  /** адрес клиента веб-сессии */
  source?: string
  /** сервисный режим: команда и кластер, к которым относится действие */
  teamId?: string
  clusterId?: string
}

/** Условия выборки (internal/audit.Filter) */
export interface AuditFilter {
  user: string
  action: string
  text: string
  result: string
  since: string
  until: string
  limit: number
  /** идентификатор команды / кластера (сервисный режим); '' — любые */
  team: string
  cluster: string
}

/** Ответ GetAuditLog (internal/audit.Page) */
export interface AuditPage {
  events: AuditEvent[]
  truncated: boolean
  writeError: string
}

/** Идентификаторы результатов (подписи — audit.results.*) */
export const RESULT_IDS = ['ok', 'failed', 'denied'] as const

/** Подпись результата; неизвестное значение показывается как есть */
export function resultLabel(result: string): string {
  return (RESULT_IDS as readonly string[]).includes(result) ? t(`audit.results.${result}`) : result
}

// Префиксы действий вида DeleteDeployment → «Удаление: Deployment» (подписи — audit.verbs.*)
const ACTION_VERBS = ['Delete', 'Scale', 'Restart', 'Rollback', 'Retry', 'Evict']

/** Понятное название действия; неизвестные имена показываются как есть */
export function actionLabel(action: string): string {
  const key = `audit.actions.${action}`
  if (te(key)) return t(key)
  for (const prefix of ACTION_VERBS) {
    if (action.startsWith(prefix) && action.length > prefix.length) {
      return `${t(`audit.verbs.${prefix}`)}: ${action.slice(prefix.length)}`
    }
  }
  return action
}
