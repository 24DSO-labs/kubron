// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { onMounted, onUnmounted, ref, shallowRef, watch, type Ref } from 'vue'
import { callApp } from '../resources/wails'
import { can } from './useAuth'
import { Perm } from '../types/auth'
import type { CertScan, ChangeEvent, OverviewRisks } from '../types/insights'

const REFRESH_MS = 60_000
// Сертификаты меняются раз в недели, а чтение секретов — самая тяжёлая операция: проверяем редко
const CERTS_REFRESH_MS = 10 * 60_000

/**
 * Три независимых источника для главного экрана: запас по узлам / квоты / PDB, лента изменений и сертификаты.
 * Каждый обновляется отдельно, ошибка одного не прячет остальные; устаревшие ответы отбрасываются.
 * Сертификаты запрашиваются, только если у пользователя есть право secrets.read (бэкенд проверяет его сам).
 */
export function useInsights(clusterName: Ref<string>) {
  const risks = shallowRef<OverviewRisks | null>(null)
  const changes = shallowRef<ChangeEvent[] | null>(null)
  const certs = shallowRef<CertScan | null>(null)
  /** окно ленты изменений, минуты */
  const windowMinutes = ref(60)

  let seqRisks = 0, seqChanges = 0, seqCerts = 0
  const timers: ReturnType<typeof setInterval>[] = []

  async function loadRisks() {
    const my = ++seqRisks
    try {
      const r = (await callApp('GetOverviewRisks')) as unknown as OverviewRisks
      if (my === seqRisks) risks.value = r
    } catch { /* остаются прошлые данные */ }
  }
  async function loadChanges() {
    const my = ++seqChanges
    try {
      const c = (await callApp('GetRecentChanges', windowMinutes.value)) as unknown as ChangeEvent[]
      if (my === seqChanges) changes.value = c ?? []
    } catch { /* остаются прошлые данные */ }
  }
  async function loadCerts() {
    if (!can(Perm.SecretsRead)) return
    const my = ++seqCerts
    try {
      const c = (await callApp('GetCertificateExpiry')) as unknown as CertScan
      if (my === seqCerts) certs.value = c
    } catch { /* нет прав или кластер недоступен: карточка просто не покажется */ }
  }

  const visible = (f: () => void) => () => { if (!document.hidden) f() }
  function reset() {
    seqRisks++; seqChanges++; seqCerts++
    risks.value = null; changes.value = null; certs.value = null
  }
  function start() {
    void loadRisks(); void loadChanges(); void loadCerts()
  }

  onMounted(() => {
    start()
    timers.push(
      setInterval(visible(() => { void loadRisks(); void loadChanges() }), REFRESH_MS),
      setInterval(visible(() => void loadCerts()), CERTS_REFRESH_MS),
    )
  })
  onUnmounted(() => { reset(); timers.forEach(clearInterval) })
  watch(clusterName, () => { reset(); start() })
  watch(windowMinutes, () => { changes.value = null; void loadChanges() })

  return { risks, changes, certs, windowMinutes }
}
