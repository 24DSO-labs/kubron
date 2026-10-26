// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import {
  MAX_NAMESPACES, loadSelection, normalizeSelection, restoreSelection, sameSelection,
  saveSelection, summarizeSelection, toggleNamespace, type KeyValueStorage,
} from './namespaceSelection'

function memoryStorage(initial: Record<string, string> = {}): KeyValueStorage & { data: Record<string, string> } {
  const data = { ...initial }
  return {
    data,
    getItem: k => (k in data ? data[k] : null),
    setItem: (k, v) => { data[k] = v },
    removeItem: k => { delete data[k] },
  }
}

describe('normalizeSelection', () => {
  it('убирает пустые и повторы, сортирует', () => {
    expect(normalizeSelection([' b ', 'a', '', 'b', '  '])).toEqual(['a', 'b'])
  })
  it('отбрасывает имена, которых нет в кластере', () => {
    expect(normalizeSelection(['a', 'gone', 'b'], ['a', 'b', 'c'])).toEqual(['a', 'b'])
  })
  it('пустой результат = «все»', () => {
    expect(normalizeSelection(['gone'], ['a'])).toEqual([])
  })
  it('не превышает предел', () => {
    const many = Array.from({ length: MAX_NAMESPACES + 10 }, (_, i) => `ns-${String(i).padStart(3, '0')}`)
    expect(normalizeSelection(many)).toHaveLength(MAX_NAMESPACES)
  })
})

describe('toggleNamespace / sameSelection / summarizeSelection', () => {
  it('добавляет и убирает, не мутируя исходный массив', () => {
    const base = ['a']
    const added = toggleNamespace(base, 'b')
    expect(added).toEqual(['a', 'b'])
    expect(base).toEqual(['a'])
    expect(toggleNamespace(added, 'a')).toEqual(['b'])
  })
  it('не добавляет сверх предела', () => {
    const full = Array.from({ length: MAX_NAMESPACES }, (_, i) => `ns-${String(i).padStart(3, '0')}`)
    expect(toggleNamespace(full, 'extra')).toEqual(full)
  })
  it('сравнивает выборы', () => {
    expect(sameSelection(['a'], ['a'])).toBe(true)
    expect(sameSelection(['a'], ['a', 'b'])).toBe(false)
  })
  it('формирует подпись', () => {
    expect(summarizeSelection([])).toBe('All namespaces')
    expect(summarizeSelection(['kube-system'])).toBe('kube-system')
    expect(summarizeSelection(['a', 'b', 'c'])).toBe('3 namespaces')
  })
})

describe('сохранение выбора', () => {
  it('сохраняет и читает по кластеру', () => {
    const st = memoryStorage()
    saveSelection('c1', ['b', 'a'], st)
    saveSelection('c2', ['x'], st)
    expect(loadSelection('c1', st)).toEqual(['a', 'b'])
    expect(loadSelection('c2', st)).toEqual(['x'])
  })
  it('«все» удаляет запись', () => {
    const st = memoryStorage()
    saveSelection('c1', ['a'], st)
    saveSelection('c1', [], st)
    expect(Object.keys(st.data)).toHaveLength(0)
  })
  it('мусор и ошибки storage дают «все»', () => {
    expect(loadSelection('c1', memoryStorage({ 'kubron.namespaces.c1': '{oops' }))).toEqual([])
    expect(loadSelection('c1', memoryStorage({ 'kubron.namespaces.c1': '"str"' }))).toEqual([])
    expect(loadSelection('c1', memoryStorage({ 'kubron.namespaces.c1': '[1,"a",null]' }))).toEqual(['a'])
    const broken: KeyValueStorage = {
      getItem: () => { throw new Error('denied') },
      setItem: () => { throw new Error('quota') },
      removeItem: () => { throw new Error('denied') },
    }
    expect(loadSelection('c1', broken)).toEqual([])
    expect(() => saveSelection('c1', ['a'], broken)).not.toThrow()
  })
  it('без storage и без id ничего не падает', () => {
    expect(loadSelection('c1', null)).toEqual([])
    expect(loadSelection('', memoryStorage())).toEqual([])
    expect(() => saveSelection('', ['a'], memoryStorage())).not.toThrow()
  })
})

describe('restoreSelection', () => {
  const available = ['default', 'prod', 'stage']
  it('берёт сохранённое, выкидывая исчезнувшие namespace', () => {
    const st = memoryStorage()
    saveSelection('c1', ['prod', 'deleted'], st)
    expect(restoreSelection('c1', available, 'default', st)).toEqual(['prod'])
  })
  it('если сохранённое исчезло целиком — namespace по умолчанию', () => {
    const st = memoryStorage()
    saveSelection('c1', ['deleted'], st)
    expect(restoreSelection('c1', available, 'stage', st)).toEqual(['stage'])
  })
  it('нет ни сохранённого, ни доступного default — «все»', () => {
    expect(restoreSelection('c1', available, 'missing', memoryStorage())).toEqual([])
    expect(restoreSelection('c1', available, '', memoryStorage())).toEqual([])
  })
})
