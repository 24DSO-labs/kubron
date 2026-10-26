// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue'

/**
 * Ширина бокового меню: меняется перетаскиванием правой границы (вправо — шире).
 * Запоминается между запусками, как ширина панели деталей.
 */

const STORAGE_KEY = 'kubron.sidebarWidth'
export const SIDEBAR_MIN_WIDTH = 180
export const SIDEBAR_DEFAULT_WIDTH = 220
export const SIDEBAR_MAX_WIDTH = 480
/** Ширина полосы кластеров слева от меню (ClusterBar) */
const CLUSTER_BAR_WIDTH = 56
/** Сколько ширины окна всегда остаётся области контента */
export const SIDEBAR_MIN_CONTENT = 400

/** Ограничивает ширину: не уже минимума, не шире максимума и не настолько, чтобы задавить контент */
export function clampSidebarWidth(width: number, viewport: number): number {
  const max = Math.max(
    SIDEBAR_MIN_WIDTH,
    Math.min(SIDEBAR_MAX_WIDTH, viewport - CLUSTER_BAR_WIDTH - SIDEBAR_MIN_CONTENT),
  )
  if (!Number.isFinite(width)) return Math.min(SIDEBAR_DEFAULT_WIDTH, max)
  return Math.round(Math.min(Math.max(width, SIDEBAR_MIN_WIDTH), max))
}

function load(): number {
  try {
    const v = Number(localStorage.getItem(STORAGE_KEY))
    if (Number.isFinite(v) && v >= SIDEBAR_MIN_WIDTH) return v
  } catch { /* localStorage может быть недоступен */ }
  return SIDEBAR_DEFAULT_WIDTH
}

function save(width: number) {
  try {
    localStorage.setItem(STORAGE_KEY, String(width))
  } catch { /* не критично: ширина просто не запомнится */ }
}

const width = ref(load())
const viewport = ref(typeof window === 'undefined' ? 1280 : window.innerWidth)

if (typeof window !== 'undefined') {
  window.addEventListener('resize', () => { viewport.value = window.innerWidth })
}

export function useSidebarWidth() {
  /** С учётом текущего окна; сохранённое значение не трогаем, чтобы окно можно было снова расширить */
  const effectiveWidth = computed(() => clampSidebarWidth(width.value, viewport.value))

  function setWidth(w: number, persist = false) {
    width.value = clampSidebarWidth(w, viewport.value)
    if (persist) save(width.value)
  }

  return { width: effectiveWidth, setWidth }
}