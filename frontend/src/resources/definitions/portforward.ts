import type { k8s as models } from '../../../wailsjs/go/models'
import { defineResource } from '../types'
import { Perm } from '../../types/auth'
import { callApp } from '../wails'
import { useK8s } from '../../composables/useK8s'
import { flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'
import { BrowserOpenURL } from '../../../wailsjs/runtime/runtime'

const k8s = useK8s()

export const portForward = defineResource<models.PortForwardSession>({
  id: 'portforward',
  kind: 'Port Forward',
  view: 'portforwarding',
  title: 'Port Forwarding',
  namespaced: false,
  nav: { group: 'network', label: 'Port Forwarding' },
  autoRefresh: true,
  // локальные сессии приложения, не ресурс k8s: без YAML и удаления
  editable: false,

  columns: [
    { key: 'target', label: 'Target', className: 'resource-name' },
    { ...flatNsCol, link: (pf: any) => pf.namespace ? { id: 'namespace', ns: '', name: pf.namespace } : null },
    { key: 'localPort', label: 'Local Port', formatter: (pf: any) => `localhost:${pf.localPort}` },
    { key: 'podPort', label: 'Remote Port' },
    { key: 'status', label: 'Status', colorFn: pf => statusColor(pf.status) },
  ],

  properties: [
    { label: 'Target', value: (pf: any) => pf.target },
    { label: 'Namespace', value: pf => pf.namespace, link: (pf: any) => pf.namespace ? { id: 'namespace', ns: '', name: pf.namespace } : null },
    { label: 'Pod', value: pf => pf.pod, link: (pf: any) => ({ id: 'pod', ns: pf.namespace, name: pf.pod }) },
    {
      label: 'Service',
      value: (pf: any) => (pf.service ? `${pf.service}:${pf.servicePort}` : '—'),
      link: (pf: any) => (pf.service ? { id: 'service', ns: pf.namespace, name: pf.service } : null),
    },
    { label: 'Local Port', value: pf => pf.localPort },
    { label: 'Remote Port', value: pf => pf.podPort },
    { label: 'Status', value: pf => pf.status, badge: pf => badgeClass(pf.status) },
    { label: 'Error', value: (pf: any) => pf.error || '—' },
  ],

  actions: [
    {
      id: 'open',
      permission: Perm.PortForward,
      label: 'Open in browser',
      icon: 'open-browser',
      when: (pf: any) => pf.status === 'active',
      run: (pf: any) => BrowserOpenURL(`http://localhost:${pf.localPort}`),
    },
    {
      id: 'stop',
      permission: Perm.PortForward,
      label: 'Stop',
      icon: 'stop',
      danger: true,
      run: (pf: any, ctx) =>
        ctx.perform(
          () => callApp('StopPortForward', pf.id),
          `Port forward :${pf.localPort} → ${pf.target} остановлен`,
        ),
    },
  ],

  // Сессии запускают и из панелей pod/service: список обновляется сам
  refreshOn: () => k8s.portForwards.value,

  api: {
    list: async () => {
      await k8s.refreshPortForwards()
      return [...k8s.portForwards.value]
    },
    // «Удалить» сессию = остановить туннель и убрать запись; name сессии равен её id
    remove: (_ns, name) => callApp('StopPortForward', name),
  },
})