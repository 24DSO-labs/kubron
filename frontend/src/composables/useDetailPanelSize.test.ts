// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import {
  clampDetailWidth, DETAIL_DEFAULT_WIDTH, DETAIL_MIN_LEFTOVER, DETAIL_MIN_WIDTH,
} from './useDetailPanelSize'

describe('clampDetailWidth', () => {
  it('оставляет допустимое значение', () => {
    expect(clampDetailWidth(700, 1600)).toBe(700)
  })

  it('не уже минимума', () => {
    expect(clampDetailWidth(100, 1600)).toBe(DETAIL_MIN_WIDTH)
  })

  it('не шире окна за вычетом остатка слева', () => {
    expect(clampDetailWidth(5000, 1600)).toBe(1600 - DETAIL_MIN_LEFTOVER)
  })

  it('в узком окне минимум важнее остатка слева', () => {
    expect(clampDetailWidth(900, 500)).toBe(DETAIL_MIN_WIDTH)
  })

  it('нечисловое значение даёт ширину по умолчанию (в пределах окна)', () => {
    expect(clampDetailWidth(NaN, 1600)).toBe(DETAIL_DEFAULT_WIDTH)
    expect(clampDetailWidth(NaN, 600)).toBe(Math.max(DETAIL_MIN_WIDTH, 600 - DETAIL_MIN_LEFTOVER))
  })

  it('округляет до целых пикселей', () => {
    expect(clampDetailWidth(612.6, 1600)).toBe(613)
  })
})
