// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref } from 'vue'
import { callApp } from '../resources/wails'
import { startArgoLive, stopArgoLive } from './useArgoLive'

/** Установлен ли Argo CD (CRD Application) в активном кластере; от него зависит видимость меню */
const argoInstalled = ref(false)

/** Перепроверяет кластер; ошибка (нет прав, кластер недоступен) = «не установлен» */
async function refreshArgo(): Promise<void> {
  try {
    argoInstalled.value = await callApp('ArgoInstalled')
  } catch {
    argoInstalled.value = false
  }
  // События argocd:changed нужны только пока в кластере есть Argo CD
  if (argoInstalled.value) startArgoLive()
  else stopArgoLive()
}

function resetArgo() {
  argoInstalled.value = false
  stopArgoLive()
}

export function useArgo() {
  return { argoInstalled, refreshArgo, resetArgo }
}
