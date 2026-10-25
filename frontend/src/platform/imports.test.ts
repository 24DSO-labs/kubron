import { describe, it, expect } from 'vitest'

// ADR-0010: прямой доступ к Wails-биндингам и runtime — только из src/platform/.
// Остальной код ходит через platform/api (callApp), platform/events (onEvent), platform/files (openUrl).
// Типы из wailsjs/go/models разрешены: это общий контракт данных и для десктопа, и для веба.
const sources = import.meta.glob<string>('/src/**/*.{ts,vue}', {
  query: '?raw',
  import: 'default',
  eager: true,
})

const FORBIDDEN = [/wailsjs\/go\/main\//, /wailsjs\/runtime\//]
const IMPORT_RE = /(?:from\s+|import\s*\(\s*|import\s+)['"]([^'"]+)['"]/g

describe('platform boundary', () => {
  it('нет прямых импортов wailsjs/go/main и wailsjs/runtime вне src/platform', () => {
    const files = Object.entries(sources)
    expect(files.length).toBeGreaterThan(50) // glob действительно что-то нашёл

    const offenders: string[] = []
    for (const [path, text] of files) {
      if (path.startsWith('/src/platform/') || path.endsWith('.test.ts')) continue
      for (const m of text.matchAll(IMPORT_RE)) {
        if (FORBIDDEN.some(re => re.test(m[1]))) offenders.push(`${path}: ${m[1]}`)
      }
    }
    expect(offenders).toEqual([])
  })
})
