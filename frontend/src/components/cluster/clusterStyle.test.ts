// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import {
  CLUSTER_COLORS, CLUSTER_ENVIRONMENTS, CLUSTER_GLYPHS,
  clusterInitials, findEnvironment, findGlyph, isValidNamespace, withAlpha,
} from './clusterStyle'

describe('clusterInitials', () => {
  it('два слова — по первой букве каждого', () => {
    expect(clusterInitials('nemo dev')).toBe('ND')
  })
  it('одно слово — две первые буквы', () => {
    expect(clusterInitials('production')).toBe('PR')
  })
  it('разделители любого вида', () => {
    expect(clusterInitials('k8s-prod-eu')).toBe('KP')
    expect(clusterInitials('  stage_cluster ')).toBe('SC')
  })
  it('кириллица', () => {
    expect(clusterInitials('боевой кластер')).toBe('БК')
  })
  it('однобуквенное название', () => {
    expect(clusterInitials('a')).toBe('A')
  })
  it('нет букв и цифр — вопросительный знак', () => {
    expect(clusterInitials('')).toBe('?')
    expect(clusterInitials('--- ///')).toBe('?')
  })
})

describe('withAlpha', () => {
  it('добавляет канал прозрачности', () => {
    expect(withAlpha('#7aa2f7', 1)).toBe('#7aa2f7ff')
    expect(withAlpha('#7aa2f7', 0)).toBe('#7aa2f700')
    expect(withAlpha('#7aa2f7', 0.5)).toBe('#7aa2f780')
  })
  it('ограничивает прозрачность диапазоном 0…1', () => {
    expect(withAlpha('#000000', 5)).toBe('#000000ff')
    expect(withAlpha('#000000', -1)).toBe('#00000000')
  })
  it('чужой формат цвета не трогает', () => {
    expect(withAlpha('red', 0.5)).toBe('red')
    expect(withAlpha('#abc', 0.5)).toBe('#abc')
  })
})

describe('isValidNamespace', () => {
  it('пустое значение означает «все namespaces»', () => {
    expect(isValidNamespace('')).toBe(true)
  })
  it('принимает DNS-1123 метки', () => {
    expect(isValidNamespace('kube-system')).toBe(true)
    expect(isValidNamespace('a')).toBe(true)
    expect(isValidNamespace('team-42')).toBe(true)
  })
  it('отклоняет недопустимые имена', () => {
    expect(isValidNamespace('Apps')).toBe(false)
    expect(isValidNamespace('-apps')).toBe(false)
    expect(isValidNamespace('apps-')).toBe(false)
    expect(isValidNamespace('my_ns')).toBe(false)
    expect(isValidNamespace('a'.repeat(64))).toBe(false)
  })
})

describe('справочники оформления', () => {
  it('значок находится по ключу, устаревшие эмодзи — нет', () => {
    expect(findGlyph('server')?.label).toBe('cluster.glyph.server')
    expect(findGlyph('🔵')).toBeNull()
    expect(findGlyph('')).toBeNull()
  })
  it('окружение находится по ключу, пустое и неизвестное — нет', () => {
    expect(findEnvironment('prod')?.label).toBe('cluster.env.prod')
    expect(findEnvironment('')).toBeNull()
    expect(findEnvironment('qa')).toBeNull()
  })
  it('ключи уникальны, цвета — #rrggbb', () => {
    const glyphKeys = CLUSTER_GLYPHS.map(g => g.key)
    expect(new Set(glyphKeys).size).toBe(glyphKeys.length)
    const envKeys = CLUSTER_ENVIRONMENTS.map(e => e.key)
    expect(new Set(envKeys).size).toBe(envKeys.length)
    for (const c of [...CLUSTER_COLORS, ...CLUSTER_ENVIRONMENTS.map(e => e.color)]) {
      expect(c).toMatch(/^#[0-9a-f]{6}$/)
    }
  })
  it('ключи значков и окружений допустимы для бэкенда', () => {
    for (const g of CLUSTER_GLYPHS) expect(g.key).toMatch(/^[a-z0-9-]{1,24}$/)
    for (const e of CLUSTER_ENVIRONMENTS) expect(e.key).toMatch(/^(dev|test|stage|prod)?$/)
  })
})
