// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../platform/events', () => ({ onEvent: vi.fn(() => () => {}) }))

import { handlePartialList, resetListFailures, useListFailures } from './useListFailures'

const payload = (...ns: string[]) => ({
  method: 'GetPods',
  failures: ns.map(namespace => ({ namespace, error: `forbidden: ${namespace}` })),
})

describe('handlePartialList', () => {
  beforeEach(() => resetListFailures())

  it('показывает один тост на набор и не дублирует при автообновлении', () => {
    const notify = vi.fn()
    expect(handlePartialList(payload('b', 'a'), notify)).toBe(true)
    expect(handlePartialList(payload('a', 'b'), notify)).toBe(false) // тот же набор, другой порядок
    expect(notify).toHaveBeenCalledTimes(1)
    expect(notify.mock.calls[0][0]).toContain('a, b')
  })

  it('другой набор отказов — новый тост', () => {
    const notify = vi.fn()
    handlePartialList(payload('a'), notify)
    handlePartialList(payload('a', 'c'), notify)
    expect(notify).toHaveBeenCalledTimes(2)
  })

  it('после сброса тост показывается снова', () => {
    const notify = vi.fn()
    handlePartialList(payload('a'), notify)
    resetListFailures()
    handlePartialList(payload('a'), notify)
    expect(notify).toHaveBeenCalledTimes(2)
  })

  it('запоминает ошибки по namespace для значка «нет доступа»', () => {
    handlePartialList(payload('a'), () => {})
    expect(useListFailures().failedNamespaces.value).toEqual({ a: 'forbidden: a' })
  })

  it('пустой и некорректный payload игнорируется', () => {
    const notify = vi.fn()
    expect(handlePartialList(undefined, notify)).toBe(false)
    expect(handlePartialList({ method: 'x', failures: [] }, notify)).toBe(false)
    expect(notify).not.toHaveBeenCalled()
  })

  it('длинный список сокращается', () => {
    const notify = vi.fn()
    handlePartialList(payload('a', 'b', 'c', 'd', 'e'), notify)
    expect(notify.mock.calls[0][0]).toContain('и ещё 2')
  })
})
