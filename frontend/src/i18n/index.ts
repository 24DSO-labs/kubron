// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Локализация интерфейса (vue-i18n v11, Composition API).
 *
 * Приложение «говорит» по-английски: en — базовый язык и fallback, все остальные — переводы
 * с английского. Сообщения лежат в locales/<язык>/<раздел>.json; имя файла — первый сегмент
 * ключа (locales/en/auth.json → `auth.*`). Английский вшит в бандл, остальные языки
 * подгружаются при выборе. Правила — docs/i18n.md.
 */
import { createI18n } from 'vue-i18n'
import { DEFAULT_LOCALE, LANGUAGES, isSupportedLocale, matchLocale } from './languages'

// Структура сообщений проверяется скриптом i18n-check, а не типами TS: схема задаётся JSON-файлами
// eslint-disable-next-line @typescript-eslint/no-explicit-any
type Messages = Record<string, any>

const STORAGE_KEY = 'kubron.locale'

// en — всегда в бандле (fallback); остальные языки — отдельными чанками по требованию.
const baseFiles = import.meta.glob('./locales/en/*.json', { eager: true, import: 'default' }) as Record<string, Messages>
const lazyFiles = import.meta.glob(['./locales/*/*.json', '!./locales/en/*.json'], { import: 'default' }) as Record<
  string,
  () => Promise<Messages>
>

/** './locales/ru/auth.json' → { locale: 'ru', namespace: 'auth' } */
function parsePath(path: string): { locale: string; namespace: string } {
  const parts = path.split('/')
  return { locale: parts[parts.length - 2], namespace: parts[parts.length - 1].replace(/\.json$/, '') }
}

function mergeNamespaces(entries: Iterable<[string, Messages]>): Messages {
  const out: Messages = {}
  for (const [path, content] of entries) out[parsePath(path).namespace] = content
  return out
}

async function loadMessages(locale: string): Promise<Messages> {
  if (locale === DEFAULT_LOCALE) return mergeNamespaces(Object.entries(baseFiles))
  const files = Object.entries(lazyFiles).filter(([path]) => parsePath(path).locale === locale)
  const loaded = await Promise.all(files.map(async ([path, load]) => [path, await load()] as [string, Messages]))
  return mergeNamespaces(loaded)
}

const pluralRules = Object.fromEntries(
  LANGUAGES.filter((l) => l.pluralRule).map((l) => [l.code, l.pluralRule!]),
)

const initialMessages: Messages = { [DEFAULT_LOCALE]: mergeNamespaces(Object.entries(baseFiles)) }

export const i18n = createI18n({
  legacy: false,
  globalInjection: true, // $t в шаблонах
  // `as string`: иначе TS выводит литерал 'en' и сужает набор допустимых языков до одного
  locale: DEFAULT_LOCALE as string,
  fallbackLocale: DEFAULT_LOCALE as string,
  messages: initialMessages,
  pluralRules,
  // Пропущенный ключ в dev — предупреждение в консоли; в продакшене тихо показываем английский.
  missingWarn: import.meta.env.DEV,
  fallbackWarn: import.meta.env.DEV,
})

/**
 * Перевод вне компонентов (composables, resources/*.ts, тосты). Функции композера не зависят от
 * `this`, поэтому их можно деструктурировать. Вызов внутри computed/render реактивен к смене языка.
 */
export const { t, te, n, d } = i18n.global

/** Текущий язык (реактивный) */
export const locale = i18n.global.locale

/**
 * «Перевести, если это ключ». Для данных-описаний (подписи колонок, пунктов меню), где в поле
 * лежит ключ сообщения, а пользовательские значения (имена CRD и т. п.) должны пройти как есть.
 */
export function tl(labelOrKey: string): string {
  // Наличие ключа проверяем по английскому (он полный): в непереведённом языке t() сам откатится на en
  return te(labelOrKey, DEFAULT_LOCALE) ? t(labelOrKey) : labelOrKey
}

/** Язык из сохранённого выбора → языка браузера/системы → английский */
export function detectLocale(): string {
  try {
    const saved = localStorage.getItem(STORAGE_KEY)
    if (isSupportedLocale(saved)) return saved
  } catch {
    // localStorage недоступен (приватный режим, тесты) — идём дальше
  }
  const preferred = typeof navigator !== 'undefined' ? (navigator.languages ?? [navigator.language]) : []
  for (const tag of preferred) {
    const match = matchLocale(tag)
    if (match) return match
  }
  return DEFAULT_LOCALE
}

/** Переключить язык: подгрузить переводы, обновить <html lang> и запомнить выбор */
export async function setLocale(code: string, options: { persist?: boolean } = {}): Promise<void> {
  const target = isSupportedLocale(code) ? code : DEFAULT_LOCALE
  if (!i18n.global.availableLocales.includes(target)) {
    i18n.global.setLocaleMessage(target, await loadMessages(target))
  }
  i18n.global.locale.value = target
  if (typeof document !== 'undefined') document.documentElement.lang = target
  if (options.persist !== false) {
    try {
      localStorage.setItem(STORAGE_KEY, target)
    } catch {
      // выбор не сохранится до следующего запуска — не критично
    }
  }
}

/** Вызывается один раз до монтирования приложения */
export async function initI18n(): Promise<void> {
  await setLocale(detectLocale(), { persist: false })
}
