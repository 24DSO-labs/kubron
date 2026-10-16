import { watch } from 'vue'
import { defineResource } from '../types'
import { callApp } from '../wails'
import { useK8s } from '../../composables/useK8s'
import { reloadActive } from '../store'
import { flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'

const k8s = useK8s()

export const portForward = defineResource({
  id: 'portforward',
  kind: 'Port Forward',
  view: 'portforwarding',
  title: 'Port Forwarding',
  namespaced: false,
  // локальные сессии приложения, не ресурс k8s: без YAML и удаления
  editable: false,

  columns: [
    { key: 'id', label: 'ID', formatter: pf => String(pf.id ?? '').substring(0, 8) },
    flatNsCol,
    { key: 'pod', label: 'Pod', className: 'resource-name' },
    { key: 'localPort', label: 'Local Port' },
    { key: 'podPort', label: 'Pod Port' },
    { key: 'status', label: 'Status', colorFn: pf => statusColor(pf.status) },
  ],

  properties: [
    { label: 'Namespace', value: pf => pf.namespace },
    { label: 'Pod', value: pf => pf.pod },
    { label: 'Local Port', value: pf => pf.localPort },
    { label: 'Pod Port', value: pf => pf.podPort },
    { label: 'Status', value: pf => pf.status, badge: pf => badgeClass(pf.status) },
  ],

  actions: [
    {
      id: 'stop',
      label: 'Stop',
      danger: true,
      run: (pf, ctx) =>
        ctx.perform(
          async () => {
            await callApp('StopPortForward', pf.id)
            await k8s.refreshPortForwards()
          },
          `Port forward :${pf.localPort} → ${pf.pod}:${pf.podPort} остановлен`,
        ),
    },
  ],

  api: {
    // Список сессий живёт в useK8s (локальное состояние, не k8s API)
    list: async () => {
      await k8s.refreshPortForwards()
      return [...k8s.portForwards.value]
    },
  },
})

// Сессии меняются из других мест (панель pod): обновляем открытый список
watch(k8s.portForwards, () => {
  if (k8s.currentView.value === 'portforwarding') reloadActive()
})