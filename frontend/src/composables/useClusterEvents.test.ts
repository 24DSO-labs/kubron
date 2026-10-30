// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'

const handlers = new Map<string, (...a: any[]) => void>()
vi.mock('../platform/events', () => ({
  onEvent: vi.fn((name: string, h: (...a: any[]) => void) => { handlers.set(name, h); return () => handlers.delete(name) }),
}))

import {
  EVENT_CLUSTER_ACCESS_REVOKED, EVENT_CLUSTERS_HEALTH, handleAccessRevoked, listenClusterEvents,
} from './useClusterEvents'
import type { ClusterStatus } from '../types'

const cluster = (id: string, displayName: string) => ({ id, displayName } as ClusterStatus)

function deps(active = 'c1') {
  return {
    clusters: ref([cluster('c1', 'Prod'), cluster('c2', 'Dev')]),
    activeClusterId: ref(active),
    loadClusters: vi.fn(async () => {}),
    disconnect: vi.fn(),
    notifyError: vi.fn(),
  }
}

describe('handleAccessRevoked', () => {
  it('сбрасывает активный кластер, перечитывает список и называет кластер по имени', () => {
    const d = deps('c1')
    handleAccessRevoked('c1', d)
    expect(d.disconnect).toHaveBeenCalledTimes(1)
    expect(d.loadClusters).toHaveBeenCalledTimes(1)
    expect(d.notifyError.mock.calls[0][0]).toContain('Prod')
  })
  it('потеря не активного кластера не отключает текущий', () => {
    const d = deps('c2')
    handleAccessRevoked('c1', d)
    expect(d.disconnect).not.toHaveBeenCalled()
    expect(d.loadClusters).toHaveBeenCalled()
  })
  it('мусор вместо идентификатора не ломает обработчик', () => {
    const d = deps('c1')
    expect(() => handleAccessRevoked(undefined, d)).not.toThrow()
    expect(d.disconnect).not.toHaveBeenCalled()
  })
})

describe('listenClusterEvents', () => {
  it('перечитывает список по событию здоровья и отписывается', () => {
    handlers.clear()
    const d = deps()
    const stop = listenClusterEvents(d)
    handlers.get(EVENT_CLUSTERS_HEALTH)!()
    expect(d.loadClusters).toHaveBeenCalledTimes(1)
    handlers.get(EVENT_CLUSTER_ACCESS_REVOKED)!('c1')
    expect(d.disconnect).toHaveBeenCalledTimes(1)
    stop()
    expect(handlers.size).toBe(0)
  })
})
