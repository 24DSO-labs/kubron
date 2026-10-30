// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Реестр языков — единственное место, где нужно добавлять новые языки.
 *
 * Правила:
 *  - code  — BCP 47 код, совпадает с именем папки в locales/;
 *  - name  — название языка на самом языке (не переводится);
 *  - pluralRule — правило множественного числа (нужно для русского, арабского и др.);
 *  - any other field — можно добавить своё, languages.ts подхватит через spread.
 *
 * Чтобы добавить язык:
 *  1. создать папку locales/<code>/ с JSON-файлами;
 *  2. добавить запись сюда;
 *  3. готово.
 */

import type { PluralRule } from '../languages'

/** Русский: три формы «один | несколько | много» (1 под | 2 пода | 5 подов) */
const russianPlural: PluralRule = (choice) => {
  const n = Math.abs(choice) % 100
  const last = n % 10
  if (n > 10 && n < 20) return 2
  if (last === 1) return 0
  if (last >= 2 && last <= 4) return 1
  return 2
}

export const LANGUAGE_REGISTRY: Record<string, { name: string; pluralRule?: PluralRule }> = {
  en: { name: 'English' },
  ru: { name: 'Русский', pluralRule: russianPlural },
  cn: { name: '中文' },
}
