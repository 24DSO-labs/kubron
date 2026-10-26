// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue'

/**
 * Размер боковой панели деталей: ширина, которую меняют перетаскиванием, и режим «развернуть»
 * (панель занимает всё, кроме бокового меню). Ширина запоминается между запусками.
 */

const STORAGE_KEY = 'kubron.detailPanelWidth'
export const DETAIL_MIN_WIDTH = 380
export const DETAIL_DEFAULT_WIDTH = 550
/** Сколько ширины окна всегда остаётся списку слева (иначе панель закроет его целиком) */
export const DETAIL_MIN_LEFTOVER = 160

/** Ограничивает ширину панели: не уже минимума и не шире окна за вычетом остатка слева */
export function clampDetailWidth(width: number, viewport: number): number {
  const max = Math.max(DETAIL_MIN_WIDTH, viewport - DETAIL_MIN_LEFTOVER)
  if (!Number.isFinite(width)) return Math.min(DETAIL_DEFAULT_WIDTH, max)
  return Math.round(Math.min(Math.max(width, DETAIL_MIN_WIDTH), max))
}

function load(): number {
  try {
    const v = Number(localStorage.getItem(STORAGE_KEY))
    if (Number.isFinite(v) && v >= DETAIL_MIN_WIDTH) return v
  } catch { /* localStorage может быть недоступен */ }
  return DETAIL_DEFAULT_WIDTH
}

function save(width: number) {
  try {
    localStorage.setItem(STORAGE_KEY, String(width))
  } catch { /* не критично: ширина просто не запомнится */ }
}

const width = ref(load())
const expanded = ref(false)
const viewport = ref(typeof window === 'undefined' ? 1280 : window.innerWidth)

if (typeof window !== 'undefined') {
  window.addEventListener('resize', () => { viewport.value = window.innerWidth })
}

export function useDetailPanelSize() {
  /** Ширина с учётом текущего окна; хранимое значение не трогаем, чтобы окно можно было снова расширить */
  const effectiveWidth = computed(() => clampDetailWidth(width.value, viewport.value))

  function setWidth(w: number, persist = false) {
    width.value = clampDetailWidth(w, viewport.value)
    if (persist) save(width.value)
  }

  const toggleExpanded = () => { expanded.value = !expanded.value }
  const collapse = () => { expanded.value = false }

  return { width: effectiveWidth, expanded, setWidth, toggleExpanded, collapse }
}
