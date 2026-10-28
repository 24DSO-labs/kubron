// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Реестр языков интерфейса.
 *
 * Чтобы добавить язык:
 *  1. скопировать каталог locales/en в locales/<код> и перевести значения (ключи не менять);
 *  2. добавить запись в LANGUAGES ниже (для языков со сложными формами множественного числа —
 *     ещё и pluralRule);
 *  3. запустить `npm run i18n:check` — проверит, что ключи и плейсхолдеры совпадают с en.
 */

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

/** Русский: три формы «один | несколько | много» (1 под | 2 пода | 5 подов) */
const russianPlural: PluralRule = (choice) => {
  const n = Math.abs(choice) % 100
  const last = n % 10
  if (n > 10 && n < 20) return 2
  if (last === 1) return 0
  if (last >= 2 && last <= 4) return 1
  return 2
}

export const DEFAULT_LOCALE = 'en'

export const LANGUAGES: readonly LanguageDef[] = [
  { code: 'en', name: 'English' },
  { code: 'ru', name: 'Русский', pluralRule: russianPlural },
]

export function isSupportedLocale(code: string | null | undefined): code is string {
  return !!code && LANGUAGES.some((l) => l.code === code)
}

/** 'ru-RU' → 'ru', 'pt_BR' → 'pt'; неподдерживаемый язык → undefined */
export function matchLocale(tag: string | null | undefined): string | undefined {
  if (!tag) return undefined
  const lower = tag.toLowerCase().replace('_', '-')
  if (isSupportedLocale(lower)) return lower
  const base = lower.split('-')[0]
  return isSupportedLocale(base) ? base : undefined
}
