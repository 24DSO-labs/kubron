// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../platform/api', () => ({ callApp: vi.fn() }))

import { callApp } from '../platform/api'
import { useTeams } from './useTeams'

const TEAMS = [
  { id: 'a', name: 'Alpha', members: 3, clusters: 2, myRole: 'admin' },
  { id: 'b', name: 'Beta', members: 1, clusters: 0, myRole: 'operator' },
  { id: 'c', name: 'Gamma', members: 0, clusters: 0, myRole: '' },
]

describe('useTeams', () => {
  beforeEach(() => { vi.mocked(callApp).mockReset() })

  it('загружает команды и выделяет те, где пользователь администратор', async () => {
    vi.mocked(callApp).mockResolvedValue(TEAMS as never)
    const { teams, adminTeams, loadTeams, loadError } = useTeams()
    await loadTeams()
    expect(callApp).toHaveBeenCalledWith('ListTeams')
    expect(teams.value).toHaveLength(3)
    // Добавлять кластеры можно только в команды, где пользователь администратор; не-член не считается
    expect(adminTeams.value.map(t => t.id)).toEqual(['a'])
    expect(loadError.value).toBe('')
  })

  it('при ошибке очищает список и показывает переведённый текст', async () => {
    vi.mocked(callApp).mockRejectedValue(new Error('teams are available only in service mode'))
    const { teams, loadTeams, loadError } = useTeams()
    await loadTeams()
    expect(teams.value).toEqual([])
    expect(loadError.value).not.toBe('')
  })
})
