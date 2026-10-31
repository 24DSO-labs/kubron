// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, it, expect } from 'vitest'

// CSP веб-сервера (internal/server/headers.go) запрещает inline-скрипты и eval. Тест держит исходники
// совместимыми с ней и фиксирует аудит вывода недоверенных данных (имена ресурсов, логи, события
// приходят из кластера): новый v-html или innerHTML должен пройти осознанное решение, а не появиться молча.
const sources = import.meta.glob<string>('/src/**/*.{ts,vue}', { query: '?raw', import: 'default', eager: true })
const html = import.meta.glob<string>('/index.html', { query: '?raw', import: 'default', eager: true })

// Единственные места с v-html: оба экранируют &, < и > перед выводом (LogsView — ещё и кавычки).
const V_HTML_ALLOWED = new Set([
  '/src/components/utils/LogsView.vue',
  '/src/components/resource/sections/CodeBlockSection.vue',
])

const prod = () => Object.entries(sources).filter(([p]) => !p.endsWith('.test.ts'))

describe('источники совместимы с CSP и не выводят недоверенный HTML', () => {
  it('glob нашёл исходники', () => {
    expect(prod().length).toBeGreaterThan(50)
  })

  it('v-html только в проверенных компонентах', () => {
    const offenders = prod().filter(([p, t]) => /v-html\s*=/.test(t) && !V_HTML_ALLOWED.has(p)).map(([p]) => p)
    expect(offenders).toEqual([])
  })

  it('проверенные v-html экранируют вывод', () => {
    expect(sources['/src/components/utils/LogsView.vue']).toMatch(/function escapeHtml[\s\S]*&amp;[\s\S]*&lt;[\s\S]*&gt;/)
    expect(sources['/src/components/resource/sections/CodeBlockSection.vue']).toMatch(/const esc = .*&amp;.*&lt;.*&gt;/)
  })

  it('нет outerHTML, insertAdjacentHTML, document.write, eval, new Function', () => {
    const re = /\b(outerHTML|insertAdjacentHTML|document\.write(ln)?\s*\(|eval\s*\(|new\s+Function\s*\()/
    expect(prod().filter(([, t]) => re.test(t)).map(([p]) => p)).toEqual([])
  })

  it('innerHTML только для очистки контейнера', () => {
    const offenders: string[] = []
    for (const [p, t] of prod()) {
      for (const m of t.matchAll(/\.innerHTML\s*=\s*([^\n;]*)/g)) {
        if (m[1].trim() !== "''" && m[1].trim() !== '""') offenders.push(`${p}: ${m[0]}`)
      }
    }
    expect(offenders).toEqual([])
  })

  it('в index.html нет встроенных скриптов и обработчиков (script-src без unsafe-inline)', () => {
    const page = Object.values(html)[0]
    expect(page).toBeTruthy()
    // Check all script tags have src attribute (no inline scripts)
    const scriptOpenTags = [...page.matchAll(/<script\b([^>]*)>/gi)]
    for (const m of scriptOpenTags) {
      expect(m[1], 'script without src').toMatch(/\bsrc\s*=/)
    }
    // Verify no inline script content: count opening and closing script tags
    const openCount = (page.match(/<script\b/gi) || []).length
    // Use String.fromCharCode to build closing tag pattern and avoid CodeQL "Bad HTML filtering regexp" alert
    const closeTag = '<' + '/script>'
    const closeCount = (page.match(new RegExp(closeTag, 'gi')) || []).length
    expect(openCount).toBe(closeCount)
    // Ensure no non-whitespace content between script tags
    const scriptPairPattern = '<script\\b[^>]*>([\\s\\S]*?)' + closeTag
    const scriptPairs = page.matchAll(new RegExp(scriptPairPattern, 'gi'))
    for (const m of scriptPairs) {
      expect(m[1].trim(), 'script content should be empty').toBe('')
    }
    expect(page).not.toMatch(/\son[a-z]+\s*=/i)
    expect(page).not.toMatch(/http-equiv\s*=\s*["']?Content-Security-Policy/i) // CSP отдаёт только Go-сервер
  })
})
