// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/** Разделы управления, которые не зависят от кластера: открываются и без подключения к нему. */
export type ManagementViewId = 'teams' | 'users' | 'audit'

export interface ManagementView {
  view: ManagementViewId
  /** Ключ перевода названия (то же, что в боковом меню) */
  labelKey: string
}

/**
 * Какие разделы управления доступны: команды — в сервисном режиме, пользователи и журнал — по праву.
 * Нужен экрану без кластеров: новый администратор организации должен суметь создать команду и добавить
 * кластер, а боковое меню появляется только после подключения к кластеру.
 */
export function managementViews(access: { teams: boolean; users: boolean; audit: boolean }): ManagementView[] {
  const out: ManagementView[] = []
  if (access.teams) out.push({ view: 'teams', labelKey: 'nav.teams' })
  if (access.users) out.push({ view: 'users', labelKey: 'nav.users' })
  if (access.audit) out.push({ view: 'audit', labelKey: 'nav.audit' })
  return out
}

export function isManagementView(view: string): view is ManagementViewId {
  return view === 'teams' || view === 'users' || view === 'audit'
}
