import type { k8s as models } from '../../../wailsjs/go/models'
import { defineResource } from '../types'
import { callApp } from '../wails'
import { useK8s } from '../../composables/useK8s'
import { flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'

const k8s = useK8s()

export const portForward = defineResource<models.PortForwardSession>({
  id: 'portforward',
  kind: 'Port Forward',
  view: 'portforwarding',
  title: 'Port Forwarding',
  namespaced: false,
  nav: { group: 'network', label: 'Port Forwarding' },
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
      // perform() сам перезагрузит список, а list обновит сессии
      run: (pf, ctx) =>
        ctx.perform(
          () => callApp('StopPortForward', pf.id),
          `Port forward :${pf.localPort} → ${pf.pod}:${pf.podPort} остановлен`,
        ),
    },
  ],

  // Сессии запускают и из панелей pod/service: список обновляется сам
  refreshOn: () => k8s.portForwards.value,

  api: {
    // Список сессий живёт в useK8s (локальное состояние, не k8s API)
    list: async () => {
      await k8s.refreshPortForwards()
      return [...k8s.portForwards.value]
    },
  },
})
