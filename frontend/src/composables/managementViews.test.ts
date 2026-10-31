// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { isManagementView, managementViews } from './managementViews'

describe('managementViews', () => {
  it('в десктопе без прав управления разделов нет', () => {
    expect(managementViews({ teams: false, users: false, audit: false })).toEqual([])
  })
  it('администратор организации в сервисном режиме видит все три', () => {
    expect(managementViews({ teams: true, users: true, audit: true }).map(v => v.view)).toEqual(['teams', 'users', 'audit'])
  })
  it('администратор команды: команды и журнал, без пользователей', () => {
    expect(managementViews({ teams: true, users: false, audit: true }).map(v => v.view)).toEqual(['teams', 'audit'])
  })
  it('узнаёт только разделы управления', () => {
    expect(isManagementView('teams')).toBe(true)
    expect(isManagementView('pods')).toBe(false)
    expect(isManagementView('settings')).toBe(false)
  })
})
