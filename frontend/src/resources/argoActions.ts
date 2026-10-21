import { ref, shallowRef } from 'vue'
import { callApp } from './wails'
import { askConfirm } from './confirm'
import { reloadActive } from './store'
import { useNotifications } from '../composables/useNotifications'
import { useK8s } from '../composables/useK8s'
import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import type {
  ArgoActionResult, ArgoAppRef, ArgoApplicationInfo, ArgoHistoryEntry, ArgoSyncOptions,
} from '../types/argocd'

/**
 * Действия над Argo CD Application. Отдельный модуль, как confirm.ts: определения ресурсов
 * импортируют его напрямую, а не через actions.ts (иначе registry -> definitions -> actions даст цикл).
 */
const notifications = useNotifications()
const errText = (e: any): string => e?.message || String(e)

/** Диалог Sync (рисует ResourceHost). Одно приложение — обычный sync; несколько — массовый */
export const pendingArgoSync = shallowRef<{ apps: ArgoApplicationInfo[] } | null>(null)
/** Растёт после каждого действия: дашборд по нему перечитывает данные */
export const argoActionTick = ref(0)

export const requestArgoSync = (apps: ArgoApplicationInfo[]) => {
  if (apps.length) pendingArgoSync.value = { apps }
}
export const closeArgoSync = () => { pendingArgoSync.value = null }

export const refOf = (a: Pick<ArgoApplicationInfo, 'namespace' | 'name'>): ArgoAppRef => ({
  namespace: a.namespace,
  name: a.name,
})

/** Argo реагирует с задержкой: обновляем сразу и ещё раз через пару секунд */
function changed() {
  const bump = () => { argoActionTick.value++; void reloadActive() }
  bump()
  setTimeout(bump, 3000)
}

/** Тосты по результатам; true — всё прошло успешно */
function report(results: ArgoActionResult[], done: string): boolean {
  const failed = results.filter(r => r.error)
  const ok = results.length - failed.length
  if (ok) {
    notifications.notifySuccess(results.length === 1 ? `${results[0].name}: ${done}` : `${done}: ${ok} of ${results.length}`)
  }
  for (const f of failed.slice(0, 3)) notifications.notifyError(`${f.name}: ${f.error}`)
  if (failed.length > 3) notifications.notifyError(`…and ${failed.length - 3} more failed`)
  return failed.length === 0
}

export async function runArgoSync(apps: ArgoApplicationInfo[], opts: ArgoSyncOptions): Promise<void> {
  try {
    const res = await callApp('SyncArgoApplication', apps.map(refOf), opts)
    report(res, opts.dryRun ? 'dry-run started' : 'sync started')
    changed()
  } catch (e) {
    notifications.notifyError(`Sync: ${errText(e)}`)
  }
}

export async function refreshArgoApps(apps: ArgoApplicationInfo[], hard = false): Promise<void> {
  try {
    const res = await callApp('RefreshArgoApplication', apps.map(refOf), hard)
    report(res, hard ? 'hard refresh requested' : 'refresh requested')
    changed()
  } catch (e) {
    notifications.notifyError(`Refresh: ${errText(e)}`)
  }
}

export async function terminateArgoOperation(apps: ArgoApplicationInfo[]): Promise<void> {
  const what = apps.length === 1 ? `of ${apps[0].name}` : `of ${apps.length} applications`
  if (!(await askConfirm(`Terminate the running operation ${what}?`))) return
  try {
    const res = await callApp('TerminateArgoOperation', apps.map(refOf))
    report(res, 'operation terminating')
    changed()
  } catch (e) {
    notifications.notifyError(`Terminate: ${errText(e)}`)
  }
}

export async function rollbackArgo(app: Pick<ArgoApplicationInfo, 'namespace' | 'name'>, entry: ArgoHistoryEntry): Promise<void> {
  if (!(await askConfirm(`Roll back ${app.name} to deployment #${entry.id}? Resources will be synced to that revision (no prune).`))) return
  try {
    await callApp('RollbackArgoApplication', app.namespace, app.name, entry.id, false, false)
    notifications.notifySuccess(`${app.name}: rollback to #${entry.id} started`)
    changed()
  } catch (e) {
    notifications.notifyError(`Rollback: ${errText(e)}`)
  }
}

/** Порт-форвард на argocd-server и открытие в браузере (по самоподписанному сертификату браузер предупредит) */
export async function openArgoUI(): Promise<void> {
  try {
    const info = await callApp('OpenArgoUI')
    await useK8s().refreshPortForwards()
    BrowserOpenURL(info.url)
    if (!info.reused) notifications.notifySuccess(`Argo CD UI: ${info.url}`)
  } catch (e) {
    notifications.notifyError(`Argo CD UI: ${errText(e)}`)
  }
}