// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { LANGUAGES, isSupportedLocale, matchLocale } from './languages'

describe('matchLocale', () => {
  it('приводит региональные теги к поддерживаемому языку', () => {
    expect(matchLocale('ru-RU')).toBe('ru')
    expect(matchLocale('en_US')).toBe('en')
    expect(matchLocale('EN')).toBe('en')
  })

  it('неподдерживаемые и пустые значения → undefined', () => {
    expect(matchLocale('xx-YY')).toBeUndefined()
    expect(matchLocale('')).toBeUndefined()
    expect(matchLocale(null)).toBeUndefined()
  })

  it('isSupportedLocale знает все языки из реестра', () => {
    for (const l of LANGUAGES) expect(isSupportedLocale(l.code)).toBe(true)
    expect(isSupportedLocale('xx')).toBe(false)
  })
})

describe('множественное число (ru)', () => {
  const rule = LANGUAGES.find((l) => l.code === 'ru')!.pluralRule!
  // индекс варианта в «один | несколько | много»
  const cases: [number, number][] = [
    [1, 0], [21, 0], [101, 0],
    [2, 1], [3, 1], [4, 1], [22, 1], [104, 1],
    [0, 2], [5, 2], [11, 2], [12, 2], [14, 2], [20, 2], [111, 2], [112, 2],
  ]
  it.each(cases)('%i → форма %i', (n, form) => {
    expect(rule(n, 3)).toBe(form)
  })
})
