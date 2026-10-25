import { describe, it, expect } from 'vitest'
import { readdirSync, readFileSync, statSync } from 'node:fs'
import { join, relative, sep } from 'node:path'
import { fileURLToPath } from 'node:url'

// ADR-0010: прямой доступ к Wails-биндингам и runtime — только из src/platform/.
// Остальной код ходит через platform/api (callApp), platform/events (onEvent), platform/files (openUrl).
// Типы из wailsjs/go/models разрешены: это общий контракт данных и для десктопа, и для веба.
const SRC = join(fileURLToPath(new URL('.', import.meta.url)), '..')
const FORBIDDEN = [/wailsjs\/go\/main\//, /wailsjs\/runtime\//]
const IMPORT_RE = /(?:from\s+|import\s*\(\s*|import\s+)['"]([^'"]+)['"]/g

function* walk(dir: string): Generator<string> {
  for (const name of readdirSync(dir)) {
    const p = join(dir, name)
    if (statSync(p).isDirectory()) yield* walk(p)
    else if (/\.(ts|vue)$/.test(name) && !/\.test\.ts$/.test(name)) yield p
  }
}

describe('platform boundary', () => {
  it('нет прямых импортов wailsjs/go/main и wailsjs/runtime вне src/platform', () => {
    const offenders: string[] = []
    for (const file of walk(SRC)) {
      const rel = relative(SRC, file).split(sep).join('/')
      if (rel.startsWith('platform/')) continue
      const text = readFileSync(file, 'utf8')
      for (const m of text.matchAll(IMPORT_RE)) {
        if (FORBIDDEN.some(re => re.test(m[1]))) offenders.push(`${rel}: ${m[1]}`)
      }
    }
    expect(offenders).toEqual([])
  })
})
