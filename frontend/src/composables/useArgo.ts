import { ref } from 'vue'
import { callApp } from '../resources/wails'

/** Установлен ли Argo CD (CRD Application) в активном кластере; от него зависит видимость меню */
const argoInstalled = ref(false)

/** Перепроверяет кластер; ошибка (нет прав, кластер недоступен) = «не установлен» */
async function refreshArgo(): Promise<void> {
  try {
    argoInstalled.value = await callApp('ArgoInstalled')
  } catch {
    argoInstalled.value = false
  }
}

function resetArgo() {
  argoInstalled.value = false
}

export function useArgo() {
  return { argoInstalled, refreshArgo, resetArgo }
}
