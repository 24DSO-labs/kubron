// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Реестр языков интерфейса.
 *
 * Чтобы добавить язык:
 *  1. создать папку locales/<код>/ с JSON-файлами;
 *  2. добавить запись в locales/registry.ts (название, pluralRule и т.д.).
 *
 * languages.ts автоматически сканирует папку locales/ + читает registry.ts.
 */

import { LANGUAGE_REGISTRY } from './locales/registry'

/** Правило выбора формы множественного числа: возвращает индекс варианта в «a | b | c» */
export type PluralRule = (choice: number, choicesLength: number) => number

export interface LanguageDef {
  /** BCP 47, совпадает с именем каталога в locales/ */
  code: string
  /** Название на самом языке: не переводится и не зависит от выбранного языка */
  name: string
  /** Правило множественного числа; без него vue-i18n использует правило английского (2 формы) */
  pluralRule?: PluralRule
}

export const DEFAULT_LOCALE = 'en'

/** Сканирует папку locales/ и возвращает список кодов языковых папок */
function scanLocales(): string[] {
  // VITE_LOCALES — строка вида "en,ru,cn", задаётся Vite-плагином при сборке
  const raw = import.meta.env.VITE_LOCALES
  if (raw) {
    return raw.split(',').map((s: string) => s.trim()).filter(Boolean)
  }
  // Fallback для IDE / ручных проверок — берём ключи из registry
  return Object.keys(LANGUAGE_REGISTRY).sort()
}

/** Построить LANGUAGES из обнаруженных папок + registry */
export const LANGUAGES: readonly LanguageDef[] = (() => {
  const codes = scanLocales()
  const out: LanguageDef[] = []

  for (const code of codes) {
    const def = LANGUAGE_REGISTRY[code]
    if (!def) continue
    out.push({ code, name: def.name, ...(def.pluralRule ? { pluralRule: def.pluralRule } : {}) })
  }

  return Object.freeze(out.map((l) => Object.freeze(l)))
})()

const isKnown = (code: string) => LANGUAGES.some((l) => l.code === code)

export function isSupportedLocale(code: string | null | undefined): code is string {
  return !!code && isKnown(code)
}

/** 'ru-RU' → 'ru', 'pt_BR' → 'pt'; неподдерживаемый язык → undefined */
export function matchLocale(tag: string | null | undefined): string | undefined {
  if (!tag) return undefined
  const lower = tag.toLowerCase().replace('_', '-')
  if (isKnown(lower)) return lower
  const base = lower.split('-')[0]
  return isKnown(base) ? base : undefined
}
