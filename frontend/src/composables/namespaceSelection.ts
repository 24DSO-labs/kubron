// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Выбор namespace: пустой массив = все, один элемент = один, несколько = набор.
 * Чистые функции без Vue и без обращений к глобальному storage — их удобно тестировать.
 */

/** Предел размера набора; совпадает с maxScopeNamespaces на стороне Go. */
export const MAX_NAMESPACES = 50

const STORAGE_PREFIX = 'kubron.namespaces.'

/**
 * Приводит выбор к каноническому виду: без пустых и повторяющихся имён, отсортированный.
 * Если передан список имён кластера (available), лишние имена отбрасываются.
 * Пустой результат означает «все».
 */
export function normalizeSelection(names: readonly string[], available?: readonly string[]): string[] {
  const known = available ? new Set(available) : null
  const out = new Set<string>()
  for (const raw of names) {
    const n = typeof raw === 'string' ? raw.trim() : ''
    if (!n || (known && !known.has(n))) continue
    out.add(n)
  }
  return [...out].sort().slice(0, MAX_NAMESPACES)
}

export function sameSelection(a: readonly string[], b: readonly string[]): boolean {
  return a.length === b.length && a.every((n, i) => n === b[i])
}

/** Добавляет или убирает namespace. Возвращает новый массив: исходный не мутируется. */
export function toggleNamespace(selection: readonly string[], name: string): string[] {
  if (selection.includes(name)) return selection.filter(n => n !== name)
  if (selection.length >= MAX_NAMESPACES) return [...selection]
  return normalizeSelection([...selection, name])
}

/** Подпись для свёрнутого селектора. */
export function summarizeSelection(selection: readonly string[]): string {
  if (selection.length === 0) return 'All namespaces'
  if (selection.length === 1) return selection[0]
  return `${selection.length} namespaces`
}

/** Минимальный интерфейс storage (подменяется в тестах). */
export type KeyValueStorage = Pick<Storage, 'getItem' | 'setItem' | 'removeItem'>

function defaultStorage(): KeyValueStorage | null {
  try { return typeof localStorage === 'undefined' ? null : localStorage } catch { return null }
}

/** Читает сохранённый выбор кластера. Любая ошибка или мусор в storage дают пустой выбор. */
export function loadSelection(clusterId: string, storage: KeyValueStorage | null = defaultStorage()): string[] {
  if (!clusterId || !storage) return []
  try {
    const raw = storage.getItem(STORAGE_PREFIX + clusterId)
    const parsed = raw ? JSON.parse(raw) : []
    return Array.isArray(parsed) ? normalizeSelection(parsed.filter((x): x is string => typeof x === 'string')) : []
  } catch {
    return []
  }
}

/** Сохраняет выбор кластера; «все» хранится как отсутствие записи. */
export function saveSelection(clusterId: string, selection: readonly string[], storage: KeyValueStorage | null = defaultStorage()): void {
  if (!clusterId || !storage) return
  try {
    if (selection.length === 0) storage.removeItem(STORAGE_PREFIX + clusterId)
    else storage.setItem(STORAGE_PREFIX + clusterId, JSON.stringify(normalizeSelection(selection)))
  } catch { /* storage недоступен или переполнен: выбор просто не запомнится */ }
}

/**
 * Выбор после загрузки списка namespace кластера: сохранённый (без исчезнувших namespace),
 * иначе namespace по умолчанию, если он есть в кластере, иначе «все».
 */
export function restoreSelection(
  clusterId: string,
  available: readonly string[],
  defaultNamespace: string,
  storage: KeyValueStorage | null = defaultStorage(),
): string[] {
  const saved = normalizeSelection(loadSelection(clusterId, storage), available)
  if (saved.length) return saved
  return defaultNamespace && available.includes(defaultNamespace) ? [defaultNamespace] : []
}
