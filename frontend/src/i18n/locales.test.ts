// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { execFileSync } from 'node:child_process'
import { fileURLToPath } from 'node:url'
import { describe, expect, it } from 'vitest'

// Полнота локалей, плейсхолдеры, ссылки на ключи и кириллица в коде — в scripts/i18n-check.mjs;
// тот же скрипт запускается как `npm run i18n:check`, здесь он только привязан к `npm test`.
describe('локали', () => {
  it('i18n-check проходит без ошибок', () => {
    const script = fileURLToPath(new URL('../../../scripts/i18n-check.mjs', import.meta.url))
    let output = ''
    try {
      output = execFileSync(process.execPath, [script], { encoding: 'utf8' })
    } catch (e) {
      output = (e as { stdout?: string }).stdout ?? String(e)
      throw new Error(output)
    }
    expect(output).toContain('i18n-check: OK')
  })
})
