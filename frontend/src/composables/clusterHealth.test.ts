// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { clusterTitle, healthLevel, healthText } from './clusterHealth'
import type { ClusterHealth } from '../types'

const h = (state: ClusterHealth['state'], message = ''): ClusterHealth => ({ state, message, checkedAt: '2026-01-01T00:00:00Z' })

describe('healthLevel', () => {
  it('различает исправный, недоступный и непроверенный', () => {
    expect(healthLevel(h('ok'))).toBe('ok')
    for (const s of ['unreachable', 'unauthorized', 'misconfigured'] as const) expect(healthLevel(h(s))).toBe('bad')
    expect(healthLevel(h('unknown'))).toBe('unknown')
    expect(healthLevel(undefined)).toBe('unknown')
    expect(healthLevel(null)).toBe('unknown')
  })
  it('незнакомое состояние от новой версии сервера не считается ошибкой', () => {
    expect(healthLevel({ state: 'quantum' } as unknown as ClusterHealth)).toBe('unknown')
  })
})

describe('healthText и clusterTitle', () => {
  it('для исправного и непроверенного кластера текста нет', () => {
    expect(healthText(h('ok'))).toBe('')
    expect(healthText(undefined)).toBe('')
    expect(clusterTitle('prod', { health: h('ok') })).toBe('prod')
  })
  it('для недоступного добавляет причину второй строкой', () => {
    const text = healthText(h('unreachable', 'dial tcp 10.0.0.1:6443: i/o timeout'))
    expect(text).toContain('i/o timeout')
    expect(clusterTitle('prod', { health: h('unreachable', 'x') })).toBe(`prod\n${healthText(h('unreachable', 'x'))}`)
  })
  it('без причины показывает только название состояния', () => {
    expect(healthText(h('unauthorized'))).not.toContain(':')
  })
})
