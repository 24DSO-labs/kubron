#!/usr/bin/env node
// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Работа с CHANGELOG.md (формат Keep a Changelog) без внешних зависимостей.
 *
 *   node scripts/changelog.mjs notes <версия> [--fallback-unreleased] [--file путь]
 *       печатает текст раздела «## [версия]» без заголовка; код 1, если раздела нет или он пуст.
 *       С --fallback-unreleased при отсутствии раздела берётся «## [Unreleased]» (для rc-тегов).
 *   node scripts/changelog.mjs has <версия> [--file путь]
 *       код 0, если раздел версии есть и не пуст
 *   node scripts/changelog.mjs roll <версия> [YYYY-MM-DD] [--file путь]
 *       переносит содержимое «## [Unreleased]» в новый раздел версии, оставляет пустой Unreleased
 *       и обновляет ссылки внизу файла; код 1, если в Unreleased нет изменений
 */
import { readFileSync, writeFileSync } from 'node:fs'

const SEMVER = /^\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$/
const HEADING = /^ {0,3}##\s+\[([^\]]+)\](?:\s+-\s+(.+?))?\s*$/
const LINKREF = /^\[([^\]]+)\]:\s*(\S+)\s*$/

function fail(message) {
  console.error(`changelog: ${message}`)
  process.exit(1)
}

/** Разбор файла: индексы заголовков разделов и начало блока ссылок в конце файла. */
function parse(text) {
  const lines = text.replace(/\r\n/g, '\n').split('\n')
  if (lines.at(-1) === '') lines.pop()
  let linkStart = lines.length
  while (linkStart > 0 && (lines[linkStart - 1].trim() === '' || LINKREF.test(lines[linkStart - 1]))) linkStart--
  // линии между последним разделом и блоком ссылок остаются в разделе; пустые хвостовые отбрасываем позже
  const sections = []
  lines.forEach((line, i) => {
    const m = i < linkStart ? HEADING.exec(line) : null
    if (m) sections.push({ name: m[1], index: i, date: m[2] })
  })
  sections.forEach((s, k) => {
    s.end = k + 1 < sections.length ? sections[k + 1].index : linkStart
  })
  return { lines, sections, linkStart }
}

function bodyOf(parsed, section) {
  const body = parsed.lines.slice(section.index + 1, section.end)
  while (body.length && body[0].trim() === '') body.shift()
  while (body.length && body.at(-1).trim() === '') body.pop()
  return body
}

function find(parsed, name) {
  return parsed.sections.find((s) => s.name.toLowerCase() === name.toLowerCase())
}

function args(argv) {
  const positional = []
  const flags = { file: 'CHANGELOG.md', fallback: false }
  for (let i = 0; i < argv.length; i++) {
    if (argv[i] === '--file') flags.file = argv[++i]
    else if (argv[i] === '--fallback-unreleased') flags.fallback = true
    else positional.push(argv[i])
  }
  return { positional, flags }
}

function main() {
  const { positional, flags } = args(process.argv.slice(2))
  const [command, version, date] = positional
  if (!command || !version) fail('использование: changelog.mjs notes|has|roll <версия> [...]')
  if (!SEMVER.test(version)) fail(`некорректная версия «${version}» (ожидается X.Y.Z или X.Y.Z-rc.1)`)

  const text = readFileSync(flags.file, 'utf8')
  const parsed = parse(text)

  if (command === 'notes' || command === 'has') {
    const section = find(parsed, version)
    let body = section ? bodyOf(parsed, section) : []
    if (!body.length && flags.fallback && command === 'notes') {
      const unreleased = find(parsed, 'Unreleased')
      body = unreleased ? bodyOf(parsed, unreleased) : []
    }
    if (!body.length) fail(`в ${flags.file} нет непустого раздела [${version}]`)
    if (command === 'notes') process.stdout.write(body.join('\n') + '\n')
    return
  }

  if (command === 'roll') {
    const day = date ?? new Date().toISOString().slice(0, 10)
    if (!/^\d{4}-\d{2}-\d{2}$/.test(day)) fail(`некорректная дата «${day}» (ожидается YYYY-MM-DD)`)
    if (find(parsed, version)) fail(`раздел [${version}] уже есть`)
    const unreleased = find(parsed, 'Unreleased')
    if (!unreleased) fail('нет раздела [Unreleased]')
    const body = bodyOf(parsed, unreleased)
    if (!body.length) fail('в [Unreleased] нет изменений: сначала опишите их в CHANGELOG.md')

    const before = parsed.lines.slice(0, unreleased.index + 1)
    const after = parsed.lines.slice(unreleased.end, parsed.linkStart)
    while (after.length && after.at(-1).trim() === '') after.pop()
    const links = parsed.lines.slice(parsed.linkStart).filter((l) => l.trim() !== '')

    // ссылки: [Unreleased] смотрит от новой версии, для новой версии добавляется сравнение с прежней
    const unrelIdx = links.findIndex((l) => LINKREF.exec(l)?.[1].toLowerCase() === 'unreleased')
    if (unrelIdx >= 0) {
      const url = LINKREF.exec(links[unrelIdx])[2]
      const base = url.replace(/\/compare\/.*$/, '').replace(/\/releases\/.*$/, '')
      const prev = /\/compare\/v?([^.]+\.[^.]+\.[^./]+?)\.\.\.HEAD$/.exec(url)?.[1] ?? parsed.sections.find((s) => SEMVER.test(s.name))?.name
      links[unrelIdx] = `[Unreleased]: ${base}/compare/v${version}...HEAD`
      links.splice(unrelIdx + 1, 0, prev ? `[${version}]: ${base}/compare/v${prev}...v${version}` : `[${version}]: ${base}/releases/tag/v${version}`)
    }

    const out = [...before, '', `## [${version}] - ${day}`, '', ...body]
    if (after.length) out.push('', ...after)
    if (links.length) out.push('', ...links)
    writeFileSync(flags.file, out.join('\n') + '\n')
    return
  }

  fail(`неизвестная команда «${command}»`)
}

main()
