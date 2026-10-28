// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join } from 'node:path'
import { fileURLToPath } from 'node:url'
import { beforeAll, describe, expect, it } from 'vitest'
import { i18n, setLocale } from './index'
import {
  ARGO_TOKEN_NOT_SET_MESSAGE,
  NOT_AUTHENTICATED_MESSAGE,
  isArgoTokenMissing,
  isNotAuthenticated,
  resetBackendErrorCache,
  translateBackendError,
} from './backendErrors'

const repoRoot = fileURLToPath(new URL('../../../', import.meta.url))

describe('перевод ошибок бэкенда', () => {
  beforeAll(async () => {
    await setLocale('ru', { persist: false })
    resetBackendErrorCache()
  })

  it('переводит простое сообщение', () => {
    expect(translateBackendError(NOT_AUTHENTICATED_MESSAGE)).toBe('требуется вход в систему')
  })

  it('переводит префикс составной ошибки и оставляет причину как есть', () => {
    expect(translateBackendError('failed to save token: permission denied')).toBe('не удалось сохранить токен: permission denied')
  })

  it('переводит вложенные ошибки по частям', () => {
    const out = translateBackendError('failed to load kubeconfig: failed to read journal: open x: no such file')
    expect(out).toBe('не удалось загрузить kubeconfig: не удалось прочитать журнал: open x: no such file')
  })

  it('подставляет параметры', () => {
    const out = translateBackendError('failed to apply Deployment web: boom')
    expect(out).toContain('Deployment')
    expect(out).toContain('web')
    expect(out).toContain('boom')
    expect(out).not.toContain('failed to')
  })

  it('неизвестное сообщение возвращает как есть', () => {
    expect(translateBackendError('the server could not find the requested resource')).toBe('the server could not find the requested resource')
    expect(translateBackendError('')).toBe('')
  })

  it('на английском сообщения не меняются', async () => {
    await setLocale('en', { persist: false })
    expect(translateBackendError(NOT_AUTHENTICATED_MESSAGE)).toBe(NOT_AUTHENTICATED_MESSAGE)
    await setLocale('ru', { persist: false })
  })

  it('решения принимаются по английскому тексту, а не по переводу', () => {
    expect(isNotAuthenticated(`call failed: ${NOT_AUTHENTICATED_MESSAGE}`)).toBe(true)
    expect(isArgoTokenMissing(ARGO_TOKEN_NOT_SET_MESSAGE)).toBe(true)
    expect(isArgoTokenMissing(translateBackendError(ARGO_TOKEN_NOT_SET_MESSAGE))).toBe(false)
  })
})

/** Go-формат → шаблон таблицы: %s/%d/%q/%v/%w по порядку → {p0}, {p1}… */
function toTemplate(goFormat: string): string {
  let n = 0
  return goFormat
    .replace(/\\"/g, '"')
    .replace(/\\n/g, '\n')
    .replace(/\\\\/g, '\\')
    .replace(/%[-+# 0-9.*]*[a-zA-Z%]/g, (m) => (m === '%%' ? '%' : `{p${n++}}`))
}

function goFiles(dir: string, out: string[] = []): string[] {
  for (const name of readdirSync(dir)) {
    const full = join(dir, name)
    if (statSync(full).isDirectory()) goFiles(full, out)
    else if (name.endsWith('.go') && !name.endsWith('_test.go')) out.push(full)
  }
  return out
}

describe('Go-ошибки и таблица перевода', () => {
  it('каждая errors.New/fmt.Errorf в internal/ есть в locales/en/backend.json', () => {
    const en = JSON.parse(readFileSync(join(repoRoot, 'frontend/src/i18n/locales/en/backend.json'), 'utf8')) as Record<string, string>
    const known = new Set(Object.values(en).map((v) => v.replace(/\{'(.)'\}/g, '$1')))
    // Сообщение целиком из чужого текста: переводить нечего
    const passthrough = new Set(['{p0}'])
    const missing: string[] = []
    for (const file of goFiles(join(repoRoot, 'internal'))) {
      const text = readFileSync(file, 'utf8')
      for (const m of text.matchAll(/(?:errors\.New|fmt\.Errorf)\(\s*"((?:[^"\\]|\\.)*)"/g)) {
        const template = toTemplate(m[1])
        if (!known.has(template) && !passthrough.has(template)) missing.push(`${file.slice(repoRoot.length)}: ${m[1]}`)
      }
    }
    expect(missing, 'Новая ошибка в Go: добавьте её в backend.json (en и ru), см. docs/i18n.md').toEqual([])
  })

  it('i18n содержит английский базовый язык', () => {
    expect(i18n.global.availableLocales).toContain('en')
  })
})
