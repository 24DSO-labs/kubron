import { ref, shallowRef } from 'vue'
import { GetResourceYAML } from '../../wailsjs/go/main/App'
import { globalTabbedPanel } from '../composables/useTabbedPanel'
import { useNotifications } from '../composables/useNotifications'
import { getResourceById } from './registry'
import { reloadActive } from './store'
import { callApp } from './wails'
import { askConfirm } from './confirm'
import type { ActionContext } from './types'

const notifications = useNotifications()

/** Какая панель деталей открыта: одна на всё приложение */
export const opened = shallowRef<{ id: string; ns: string; name: string; item?: any } | null>(null)

/** Ресурс, ожидающий подтверждения удаления */
export const pendingDelete = ref<{ id: string; kind: string; ns: string; name: string } | null>(null)

/** Deployment, для которого открыт диалог масштабирования */
export const scaleTarget = shallowRef<any | null>(null)

const errMsg = (e: any) => e?.message || String(e)

export const resourceActions: ActionContext & {
  closeDetail(): void
  confirmDelete(): Promise<void>
  cancelDelete(): void
  confirmScale(replicas: number): Promise<void>
  cancelScale(): void
} = {
  open(id, ns, name, item) {
    opened.value = { id: id.toLowerCase(), ns, name, item }
  },

  closeDetail() {
    opened.value = null
  },


  async edit(id, ns, name) {
    try {
      const yaml = await GetResourceYAML(ns, id, name, true)
      globalTabbedPanel.openYamlTab(id, name, ns, yaml)
    } catch (e) {
      notifications.notifyError(`Не удалось загрузить YAML: ${errMsg(e)}`)
    }
  },

  requestDelete(id, ns, name) {
    const def = getResourceById(id)
    if (!def?.api.remove) {
      notifications.notifyError(`Удаление для ${def?.kind ?? id} не поддерживается`)
      return
    }
    pendingDelete.value = { id: def.id, kind: def.kind, ns, name }
  },

  async confirmDelete() {
    const p = pendingDelete.value
    pendingDelete.value = null
    if (!p) return
    const def = getResourceById(p.id)
    if (!def?.api.remove) return
    try {
      await def.api.remove(p.ns, p.name)
      notifications.notifySuccess(`${p.kind} "${p.name}" удалён`)
      const o = opened.value
      if (o && o.id === p.id && o.ns === p.ns && o.name === p.name) opened.value = null
      await reloadActive()
    } catch (e) {
      notifications.notifyError(`Ошибка удаления: ${errMsg(e)}`)
    }
  },

  cancelDelete() {
    pendingDelete.value = null
  },

  openLogs(ns, pod, containers, container) {
    const list = containers?.length ? containers : container ? [{ name: container }] : []
    globalTabbedPanel.openLogsTab('pod', pod, ns, list, container ?? list[0]?.name)
  },

  // Предполагается, что TabbedPanel сам подгружает поды для не-pod ресурсов (LogsTab.pods = [])
  openResourceLogs(kind, ns, name) {
    globalTabbedPanel.openLogsTab(kind, name, ns)
  },

  openShell: (ns, pod, container) =>
    globalTabbedPanel.openTerminalTab('pod', pod, ns, pod, container, 'shell'),

  openAttach: (ns, pod, container) =>
    globalTabbedPanel.openTerminalTab('pod', pod, ns, pod, container, 'attach'),

  requestScale(item, kind) {
    scaleTarget.value = kind ? { ...item, _kind: kind } : item
  },

  async confirmScale(replicas) {
    const t = scaleTarget.value
    scaleTarget.value = null
    if (!t) return
    const kind = t._kind || t.kind || 'Deployment'
    const method = kind === 'StatefulSet' ? 'ScaleStatefulSet' : 'ScaleDeployment'
    const kindLabel = kind === 'StatefulSet' ? 'StatefulSet' : 'Deployment'
    await resourceActions.perform(
      () => callApp(method, t.namespace, t.name, replicas),
      `${kindLabel} ${t.name} scaled to ${replicas} replicas`,
    )
  },

  cancelScale() {
    scaleTarget.value = null
  },

  // Свой диалог вместо window.confirm: в Wails нативный confirm работает не на всех ОС
  confirm: askConfirm,

  async perform(fn, successMessage) {
    try {
      await fn()
      if (successMessage) notifications.notifySuccess(successMessage)
      await reloadActive()
    } catch (e) {
      notifications.notifyError(errMsg(e))
    }
  },

  reload: () => reloadActive(),
  notifySuccess: m => notifications.notifySuccess(m),
  notifyError: m => notifications.notifyError(m),
}
