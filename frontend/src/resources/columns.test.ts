// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { isSortable, nextSort, parsePrefs, resolveColumns, sortItems } from './columns'

const val = (x: any) => x.v

describe('sortItems', () => {
  it('сравнивает строки «по-человечески»: pod-2 раньше pod-10', () => {
    const items = [{ v: 'pod-10' }, { v: 'pod-2' }, { v: 'pod-1' }]
    expect(sortItems(items, val, 'asc').map(val)).toEqual(['pod-1', 'pod-2', 'pod-10'])
    expect(sortItems(items, val, 'desc').map(val)).toEqual(['pod-10', 'pod-2', 'pod-1'])
  })

  it('сортирует числа как числа', () => {
    const items = [{ v: 5 }, { v: 40 }, { v: 7 }]
    expect(sortItems(items, val, 'asc').map(val)).toEqual([5, 7, 40])
    expect(sortItems(items, val, 'desc').map(val)).toEqual([40, 7, 5])
  })

  it('пустые значения всегда в конце, в любом направлении', () => {
    const items = [{ v: null }, { v: 'b' }, { v: '—' }, { v: 'a' }, { v: undefined }, { v: NaN }]
    expect(sortItems(items, val, 'asc').map(val)).toEqual(['a', 'b', null, '—', undefined, NaN])
    expect(sortItems(items, val, 'desc').map(val)).toEqual(['b', 'a', null, '—', undefined, NaN])
  })

  it('ноль не считается пустым значением', () => {
    expect(sortItems([{ v: 3 }, { v: 0 }, { v: null }], val, 'asc').map(val)).toEqual([0, 3, null])
  })

  it('стабильна: равные значения сохраняют исходный порядок в обоих направлениях', () => {
    const items = [{ v: 1, id: 'x' }, { v: 1, id: 'y' }, { v: 1, id: 'z' }]
    expect(sortItems(items, val, 'asc').map((i: any) => i.id)).toEqual(['x', 'y', 'z'])
    expect(sortItems(items, val, 'desc').map((i: any) => i.id)).toEqual(['x', 'y', 'z'])
  })

  it('не мутирует исходный массив', () => {
    const items = [{ v: 2 }, { v: 1 }]
    sortItems(items, val, 'asc')
    expect(items.map(val)).toEqual([2, 1])
  })
})

describe('nextSort', () => {
  it('asc -> desc -> без сортировки', () => {
    const asc = nextSort(null, 'age')
    expect(asc).toEqual({ key: 'age', dir: 'asc' })
    const desc = nextSort(asc, 'age')
    expect(desc).toEqual({ key: 'age', dir: 'desc' })
    expect(nextSort(desc, 'age')).toBeNull()
  })

  it('клик по другой колонке начинает с asc', () => {
    expect(nextSort({ key: 'age', dir: 'desc' }, 'name')).toEqual({ key: 'name', dir: 'asc' })
  })
})

describe('resolveColumns', () => {
  const cols = [{ key: 'name' }, { key: 'status' }, { key: 'age' }]
  const prefs = (p: Partial<{ order: string[]; hidden: string[] }>) => ({ order: [], hidden: [], sort: null, ...p })

  it('без настроек — порядок определения', () => {
    expect(resolveColumns(cols, prefs({})).visible.map(c => c.key)).toEqual(['name', 'status', 'age'])
  })

  it('применяет порядок, неизвестные ключи игнорирует, новые колонки добавляет в конец', () => {
    const r = resolveColumns(cols, prefs({ order: ['age', 'ghost', 'name', 'age'] }))
    expect(r.ordered.map(c => c.key)).toEqual(['age', 'name', 'status'])
  })

  it('скрывает колонки, но первая (имя) всегда видна', () => {
    const r = resolveColumns(cols, prefs({ hidden: ['name', 'status'] }))
    expect(r.visible.map(c => c.key)).toEqual(['name', 'age'])
    expect(r.ordered).toHaveLength(3) // в меню скрытые колонки остаются, чтобы их можно было вернуть
  })

  it('первая колонка видна, даже если пользователь переставил её', () => {
    const r = resolveColumns(cols, prefs({ order: ['age', 'name'], hidden: ['name'] }))
    expect(r.visible.map(c => c.key)).toEqual(['age', 'name', 'status'])
  })
})

describe('parsePrefs', () => {
  it('повреждённые данные дают пустые настройки', () => {
    for (const bad of [null, undefined, 42, 'x', []]) {
      expect(parsePrefs(bad)).toEqual({ order: [], hidden: [], sort: null })
    }
  })

  it('отбрасывает значения неверного типа', () => {
    expect(parsePrefs({ order: ['a', 1, null], hidden: 'b', sort: { key: 'a', dir: 'up' } }))
      .toEqual({ order: ['a'], hidden: [], sort: null })
  })

  it('читает корректные настройки', () => {
    const p = { order: ['b', 'a'], hidden: ['c'], sort: { key: 'a', dir: 'desc' as const } }
    expect(parsePrefs(p)).toEqual(p)
  })
})

describe('isSortable', () => {
  it('обычная колонка сортируется, sortable:false отключает', () => {
    expect(isSortable({})).toBe(true)
    expect(isSortable({ sortable: false })).toBe(false)
  })
  it('колонка-компонент сортируется только с sortValue', () => {
    expect(isSortable({ component: {} })).toBe(false)
    expect(isSortable({ component: {}, sortValue: () => 1 })).toBe(true)
  })
})
