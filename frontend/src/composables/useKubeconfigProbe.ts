// Разбор kubeconfig и пробное подключение для диалогов кластера.
// Бэкенд возвращает контексты без секретов, поэтому результат можно безопасно показывать.

import { ref } from 'vue'
import type { ConnectionCheck, KubeconfigInfo, KubeconfigSource } from '../types'
import { errorText } from './errorText'
import { callApp } from '../platform/api'

export const NO_KUBECONFIG_SOURCE: KubeconfigSource = { path: '', text: '', clusterId: '' }

/** Контекст для выбора по умолчанию: прежний, если он есть в kubeconfig, иначе current-context */
export function pickContextName(info: KubeconfigInfo, preferred: string): string {
  return info.contexts.some(c => c.name === preferred) ? preferred : info.currentContext
}

export function useKubeconfigProbe() {
  const info = ref<KubeconfigInfo | null>(null)
  const inspecting = ref(false)
  const inspectError = ref('')
  const check = ref<ConnectionCheck | null>(null)
  const checking = ref(false)

  // Ответы приходят асинхронно: после смены источника результат прошлого запроса должен отбрасываться.
  let inspectSeq = 0
  let checkSeq = 0

  async function inspect(src: KubeconfigSource): Promise<KubeconfigInfo | null> {
    const seq = ++inspectSeq
    inspecting.value = true
    inspectError.value = ''
    try {
      const result = await callApp('InspectKubeconfig', src)
      if (seq !== inspectSeq) return null
      info.value = result
      return result
    } catch (e) {
      if (seq !== inspectSeq) return null
      info.value = null
      inspectError.value = errorText(e)
      return null
    } finally {
      if (seq === inspectSeq) inspecting.value = false
    }
  }

  async function test(src: KubeconfigSource, contextName: string): Promise<void> {
    const seq = ++checkSeq
    checking.value = true
    check.value = null
    try {
      const result = await callApp('TestKubeconfig', src, contextName)
      if (seq === checkSeq) check.value = result
    } catch (e) {
      if (seq === checkSeq) {
        check.value = { ok: false, serverVersion: '', latencyMs: 0, warning: '', error: errorText(e) }
      }
    } finally {
      if (seq === checkSeq) checking.value = false
    }
  }

  /** Результат проверки относится к конкретным источнику и контексту — при их смене он устаревает */
  function resetCheck() {
    checkSeq++
    check.value = null
    checking.value = false
  }

  function reset() {
    inspectSeq++
    info.value = null
    inspecting.value = false
    inspectError.value = ''
    resetCheck()
  }

  return { info, inspecting, inspectError, check, checking, inspect, test, resetCheck, reset }
}
