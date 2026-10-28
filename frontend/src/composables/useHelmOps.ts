// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue'
import { onEvent } from '../platform/events'
import { callApp } from '../resources/wails'

/** Событие бэкенда с прогрессом Helm-операции (internal/k8s/helm_progress.go) */
const HELM_PROGRESS = 'helm:progress'

const MAX_LINES = 200
const AUTO_DISMISS_MS = 6000

export type HelmOpKind = 'install' | 'upgrade' | 'rollback' | 'uninstall'

export interface HelmOp {
  opId: string
  op: HelmOpKind
  namespace: string
  release: string
  phase: string
  lines: string[]
  startedAt: number
  /** Время с момента запуска по данным бэкенда (мс); интерфейс считает своё из startedAt */
  elapsedMs: number
  cancelable: boolean
  done: boolean
  error: string
}

/** Растёт при каждом завершении операции: страница журнала перечитывает историю */
export const helmJournalTick = ref(0)

export const HELM_OP_LABEL: Record<string, string> = {
  install: 'Установка',
  upgrade: 'Обновление values',
  rollback: 'Откат',
  uninstall: 'Удаление',
}

/** Активные и недавно завершённые операции (завершённые без ошибки скрываются сами) */
export const helmOps = ref<HelmOp[]>([])

let off: (() => void) | null = null

/** Идентификатор для бэкенда: [A-Za-z0-9-], 8–64 символа */
export function newOpId(): string {
  const c: Crypto | undefined = (globalThis as any).crypto
  if (c?.randomUUID) return c.randomUUID()
  return `op-${Date.now().toString(36)}-${Math.random().toString(36).slice(2, 10)}`
}

function find(opId: string): HelmOp | undefined {
  return helmOps.value.find(o => o.opId === opId)
}

function scheduleDismiss(opId: string) {
  setTimeout(() => {
    const o = find(opId)
    if (o && o.done && !o.error) dismissHelmOp(opId)
  }, AUTO_DISMISS_MS)
}

function onProgress(p: any) {
  if (!p?.opId) return
  let o = find(p.opId)
  if (!o) {
    // операция запущена не из этого окна (или событие обогнало регистрацию)
    o = {
      opId: p.opId, op: p.op, namespace: p.namespace, release: p.release, phase: '', lines: [],
      startedAt: Date.now() - (p.elapsedMs || 0), elapsedMs: 0, cancelable: !!p.cancelable, done: false, error: '',
    }
    helmOps.value.push(o)
    o = find(p.opId)!
  }
  if (p.phase) o.phase = p.phase
  if (Array.isArray(p.lines) && p.lines.length) {
    o.lines.push(...p.lines)
    if (o.lines.length > MAX_LINES) o.lines.splice(0, o.lines.length - MAX_LINES)
  }
  o.elapsedMs = p.elapsedMs ?? o.elapsedMs
  o.cancelable = !!p.cancelable
  if (p.done && !o.done) {
    o.done = true
    o.error = p.error || ''
    if (!o.error) scheduleDismiss(o.opId)
    helmJournalTick.value++
  }
}

/** Подписка идемпотентна */
export function startHelmOps() {
  if (off) return
  off = onEvent(HELM_PROGRESS, onProgress)
}

export function dismissHelmOp(opId: string) {
  const i = helmOps.value.findIndex(o => o.opId === opId)
  if (i >= 0) helmOps.value.splice(i, 1)
}

export async function cancelHelmOp(opId: string) {
  try {
    await callApp('CancelHelmOperation', opId)
  } catch {
    // уже завершилась: итоговое событие придёт само
  }
}

interface RunMeta {
  op: HelmOpKind
  namespace: string
  release: string
  cancelable?: boolean
}

/**
 * Запускает блокирующую Helm-операцию с отслеживанием прогресса: регистрирует её в панели до вызова,
 * передаёт opId бэкенду и гарантированно завершает запись, даже если бэкенд отклонил вызов
 * до начала работы (права, валидация) и итогового события нет.
 */
export async function runHelmOp<T>(meta: RunMeta, fn: (opId: string) => Promise<T>, onStart?: (opId: string) => void): Promise<T> {
  startHelmOps()
  const opId = newOpId()
  helmOps.value.push({
    opId, op: meta.op, namespace: meta.namespace, release: meta.release, phase: 'Запуск…', lines: [],
    startedAt: Date.now(), elapsedMs: 0, cancelable: !!meta.cancelable, done: false, error: '',
  })
  onStart?.(opId)
  try {
    const r = await fn(opId)
    const o = find(opId)
    if (o && !o.done) { o.done = true; scheduleDismiss(opId) }
    helmJournalTick.value++
    return r
  } catch (e: any) {
    const o = find(opId)
    if (o && !o.done) { o.done = true; o.error = e?.message || String(e) }
    helmJournalTick.value++
    throw e
  }
}
