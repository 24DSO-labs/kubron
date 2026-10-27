// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { beforeEach, describe, expect, it, vi } from 'vitest'
import { openTerminal } from './terminal'
import { callApp } from './api'
import { onEvent } from './events'

vi.mock('./api', () => ({ callApp: vi.fn() }))
vi.mock('./events', () => ({ onEvent: vi.fn() }))
vi.mock('./terminal.web', () => ({ openWebTerminal: vi.fn() }))
vi.mock('../resources/uid', () => ({ uid: () => 'u1' }))

const params = { namespace: 'ns', pod: 'p', container: 'c', shell: '', cols: 80, rows: 24 }
const handlers = () => ({ onData: vi.fn(), onExit: vi.fn() })

let listeners: Record<string, (...a: any[]) => void>
let offs: Array<ReturnType<typeof vi.fn>>

beforeEach(() => {
  vi.mocked(callApp).mockReset().mockResolvedValue(undefined as never)
  listeners = {}
  offs = []
  vi.mocked(onEvent).mockReset().mockImplementation(((name: string, fn: (...a: any[]) => void) => {
    listeners[name] = fn
    const off = vi.fn()
    offs.push(off)
    return off
  }) as never)
})

describe('openTerminal (десктоп)', () => {
  it('подписывается до старта сессии, декодирует base64 и шлёт exit', async () => {
    const h = handlers()
    let subscribedBeforeStart = false
    vi.mocked(callApp).mockImplementation((async (fn: string) => {
      if (fn === 'StartTerminalSession') subscribedBeforeStart = Object.keys(listeners).length === 2
    }) as never)
    await openTerminal(params, h)
    expect(subscribedBeforeStart).toBe(true)
    expect(callApp).toHaveBeenCalledWith('StartTerminalSession', 'term_u1', 'ns', 'p', 'c', '', 80, 24)

    listeners['terminal:data:term_u1'](btoa('hi'))
    expect(h.onData).toHaveBeenCalledWith(new Uint8Array([104, 105]))
    listeners['terminal:exit:term_u1']()
    expect(h.onExit).toHaveBeenCalledWith('exit')
  })

  it('ввод — байты UTF-8, размер и закрытие — вызовы бэкенда', async () => {
    const handle = await openTerminal(params, handlers())
    handle.write('é')
    handle.resize(100, 50)
    handle.close()
    expect(callApp).toHaveBeenCalledWith('WriteTerminal', 'term_u1', [195, 169])
    expect(callApp).toHaveBeenCalledWith('ResizeTerminal', 'term_u1', 100, 50)
    expect(callApp).toHaveBeenCalledWith('CloseTerminal', 'term_u1')
    expect(offs.every(o => o.mock.calls.length === 1)).toBe(true) // отписка при закрытии
  })

  it('ошибка старта снимает подписки', async () => {
    vi.mocked(callApp).mockRejectedValueOnce(new Error('нет прав') as never)
    await expect(openTerminal(params, handlers())).rejects.toThrow('нет прав')
    expect(offs.every(o => o.mock.calls.length === 1)).toBe(true)
  })
})
