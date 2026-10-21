import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'

export const lease = defineResource<k8s.LeaseInfo>({
  id: 'lease',
  kind: 'Lease',
  view: 'leases',
  title: 'Leases',
  namespaced: true,
  nav: { group: 'config', label: 'Leases' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (l: any) => l.namespace ? { id: 'namespace', ns: '', name: l.namespace } : null },
    { key: 'holder', label: 'Holder' },
    ageCol,
  ],

  properties: [
    { label: 'Holder', value: (l: any) => l.holder },
    { label: 'Preferred Holder', value: (l: any) => l.preferredHolder },
    { label: 'Strategy', value: (l: any) => l.strategy },
    { label: 'Lease Duration', value: (l: any) => (l.leaseDurationSeconds ? `${l.leaseDurationSeconds}s` : '') },
    { label: 'Lease Transitions', value: (l: any) => l.leaseTransitions },
    { label: 'Acquire Time', value: (l: any) => l.acquireTime },
    { label: 'Renew Time', value: (l: any) => l.renewTime },
    { label: 'Finalizers', value: (l: any) => (l.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],

  api: {
    list: ns => callApp('GetLeases', ns),
    remove: (ns, name) => callApp('DeleteResource', 'lease', ns, name),
  },
})
