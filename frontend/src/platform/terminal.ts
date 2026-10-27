// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { uid } from '../resources/uid'
import { callApp } from './api'
import { onEvent } from './events'
import { openWebTerminal } from './terminal.web'

/** Параметры терминала. shell === '' — attach к основному процессу контейнера. */
export interface TerminalParams {
  namespace: string
  pod: string
  container: string
  shell: string
  cols: number
  rows: number
}

export interface TerminalHandlers {
  /** Вывод терминала (сырые байты) */
  onData(bytes: Uint8Array): void
  /** Сессия закончилась: процесс вышел ('exit') или потеряно соединение ('lost') */
  onExit(reason: 'exit' | 'lost'): void
}

export interface TerminalHandle {
  /** Ввод пользователя (строка, как отдаёт xterm.onData) */
  write(data: string): void
  resize(cols: number, rows: number): void
  /** Закрывает сессию и завершает процесс в контейнере */
  close(): void
}

const b64ToBytes = (b64: string) => Uint8Array.from(atob(b64), c => c.charCodeAt(0))

/**
 * Открывает терминал в поде. Десктоп: id выбирает фронтенд, вывод приходит событиями Wails.
 * Веб: id выдаёт сервер, вывод и ввод идут по бинарному WebSocket (ADR-0003, ADR-0009).
 */
export async function openTerminal(p: TerminalParams, h: TerminalHandlers): Promise<TerminalHandle> {
  if (import.meta.env.VITE_TARGET === 'web') return openWebTerminal(p, h)
  return openDesktopTerminal(p, h)
}

async function openDesktopTerminal(p: TerminalParams, h: TerminalHandlers): Promise<TerminalHandle> {
  const id = `term_${uid()}`
  let closed = false
  // Подписка ДО старта сессии, чтобы не потерять первые байты (приглашение shell)
  const unsubscribe = [
    onEvent(`terminal:data:${id}`, (b64: string) => { if (!closed) h.onData(b64ToBytes(b64)) }),
    onEvent(`terminal:exit:${id}`, () => { if (!closed) h.onExit('exit') }),
  ]
  const release = () => { closed = true; unsubscribe.forEach(off => off()) }
  try {
    await callApp('StartTerminalSession', id, p.namespace, p.pod, p.container, p.shell, p.cols, p.rows)
  } catch (e) {
    release()
    throw e
  }
  return {
    write(data) {
      const bytes = Array.from(new TextEncoder().encode(data))
      callApp('WriteTerminal', id, bytes).catch(() => { })
    },
    resize(cols, rows) { callApp('ResizeTerminal', id, cols, rows).catch(() => { }) },
    close() {
      release()
      callApp('CloseTerminal', id).catch(() => { })
    },
  }
}
