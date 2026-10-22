import { describe, expect, it } from 'vitest'
import {
  clampSidebarWidth, SIDEBAR_DEFAULT_WIDTH, SIDEBAR_MAX_WIDTH, SIDEBAR_MIN_WIDTH,
} from './useSidebarWidth'

describe('clampSidebarWidth', () => {
  it('оставляет допустимое значение', () => {
    expect(clampSidebarWidth(300, 1600)).toBe(300)
  })
  it('не уже минимума', () => {
    expect(clampSidebarWidth(50, 1600)).toBe(SIDEBAR_MIN_WIDTH)
  })
  it('не шире максимума', () => {
    expect(clampSidebarWidth(2000, 2560)).toBe(SIDEBAR_MAX_WIDTH)
  })
  it('в узком окне оставляет место контенту', () => {
    // 1000 - 56 (полоса кластеров) - 400 (контент) = 544 > MAX, но при 800 лимит 344
    expect(clampSidebarWidth(500, 800)).toBe(344)
  })
  it('нечисловое значение даёт ширину по умолчанию', () => {
    expect(clampSidebarWidth(NaN, 1600)).toBe(SIDEBAR_DEFAULT_WIDTH)
  })
  it('округляет до целых пикселей', () => {
    expect(clampSidebarWidth(250.6, 1600)).toBe(251)
  })
})