// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Сортировка и пользовательские настройки колонок списков.
 * Чистая логика без Vue и Wails: покрыта тестами в columns.test.ts.
 */

export type SortDir = 'asc' | 'desc'
export interface SortState { key: string; dir: SortDir }
export interface ColumnPrefs {
  /** Порядок ключей колонок; пустой — как в определении ресурса */
  order: string[]
  /** Скрытые колонки */
  hidden: string[]
  sort: SortState | null
}
export type SortValue = string | number | null | undefined

const PREFIX = 'kubron:cols:v1:'
const collator = new Intl.Collator(undefined, { numeric: true, sensitivity: 'base' })

export const emptyPrefs = (): ColumnPrefs => ({ order: [], hidden: [], sort: null })

const strings = (v: unknown): string[] =>
  Array.isArray(v) ? v.filter((x): x is string => typeof x === 'string') : []

/** Разбор сохранённого значения; повреждённое или чужое значение даёт пустые настройки */
export function parsePrefs(raw: unknown): ColumnPrefs {
  if (!raw || typeof raw !== 'object') return emptyPrefs()
  const r = raw as Record<string, any>
  const s = r.sort
  return {
    order: strings(r.order),
    hidden: strings(r.hidden),
    sort: s && typeof s.key === 'string' && (s.dir === 'asc' || s.dir === 'desc') ? { key: s.key, dir: s.dir } : null,
  }
}

export function loadPrefs(id: string): ColumnPrefs {
  try {
    return parsePrefs(JSON.parse(localStorage.getItem(PREFIX + id) || 'null'))
  } catch {
    return emptyPrefs() // localStorage недоступен или JSON повреждён
  }
}

export function savePrefs(id: string, prefs: ColumnPrefs): void {
  try { localStorage.setItem(PREFIX + id, JSON.stringify(prefs)) } catch { /* не критично */ }
}

/**
 * Применяет настройки к колонкам определения. Неизвестные ключи из настроек отбрасываются,
 * новые колонки (которых не было при сохранении) идут в конец. Первая колонка (имя) всегда видна.
 */
export function resolveColumns<T extends { key: string }>(
  columns: readonly T[],
  prefs: ColumnPrefs,
): { ordered: T[]; visible: T[] } {
  const byKey = new Map(columns.map(c => [c.key, c]))
  const seen = new Set<string>()
  const ordered: T[] = []
  for (const k of prefs.order) {
    const c = byKey.get(k)
    if (c && !seen.has(k)) { ordered.push(c); seen.add(k) }
  }
  for (const c of columns) if (!seen.has(c.key)) ordered.push(c)

  const hidden = new Set(prefs.hidden)
  const required = columns[0]
  return { ordered, visible: ordered.filter(c => c === required || !hidden.has(c.key)) }
}

/** asc -> desc -> без сортировки; клик по другой колонке начинает с asc */
export function nextSort(cur: SortState | null, key: string): SortState | null {
  if (cur?.key !== key) return { key, dir: 'asc' }
  return cur.dir === 'asc' ? { key, dir: 'desc' } : null
}

const isEmpty = (v: SortValue): boolean =>
  v === null || v === undefined || v === '' || v === '—' || (typeof v === 'number' && Number.isNaN(v))

function compareValues(a: string | number, b: string | number): number {
  if (typeof a === 'number' && typeof b === 'number') return a - b
  return collator.compare(String(a), String(b)) // numeric: pod-2 раньше pod-10
}

/** Стабильная сортировка; пустые значения всегда в конце, независимо от направления */
export function sortItems<T>(items: readonly T[], value: (item: T) => SortValue, dir: SortDir): T[] {
  const keyed = items.map((item, i) => ({ item, i, v: value(item) }))
  keyed.sort((x, y) => {
    const xe = isEmpty(x.v)
    const ye = isEmpty(y.v)
    if (xe || ye) return xe === ye ? x.i - y.i : xe ? 1 : -1
    const c = compareValues(x.v as string | number, y.v as string | number)
    return (dir === 'asc' ? c : -c) || x.i - y.i
  })
  return keyed.map(k => k.item)
}

/** Колонку можно сортировать, если не отключено явно и значение вычислимо (ячейки-компоненты — только с sortValue) */
export function isSortable(col: { sortable?: boolean; component?: unknown; sortValue?: unknown }): boolean {
  return col.sortable !== false && (!col.component || !!col.sortValue)
}
