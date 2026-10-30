// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { authReady, authRequirement } from './clusterAuth'

describe('authRequirement', () => {
  it('в десктопе плагины работают как у kubectl: ничего не требуется', () => {
    for (const a of ['static', 'oidc-password', 'oidc-authcode', 'exec', undefined] as const) {
      expect(authRequirement(a, false)).toBe('none')
    }
  })
  it('в вебе oidc-login с паролем просит логин и пароль, остальные плагины закрыты', () => {
    expect(authRequirement('static', true)).toBe('none')
    expect(authRequirement(undefined, true)).toBe('none') // старый бэкенд без поля
    expect(authRequirement('oidc-password', true)).toBe('credentials')
    expect(authRequirement('oidc-authcode', true)).toBe('authcode')
    expect(authRequirement('exec', true)).toBe('exec')
  })
})

describe('authReady', () => {
  it('для credentials нужны и логин, и пароль', () => {
    expect(authReady('credentials', '', '')).toBe(false)
    expect(authReady('credentials', 'vasya', '')).toBe(false)
    expect(authReady('credentials', '  ', 'pw')).toBe(false)
    expect(authReady('credentials', 'vasya', 'pw')).toBe(true)
  })
  it('невозможные варианты не отправляются, обычные — всегда', () => {
    expect(authReady('none', '', '')).toBe(true)
    expect(authReady('authcode', 'v', 'p')).toBe(false)
    expect(authReady('exec', 'v', 'p')).toBe(false)
  })
})
