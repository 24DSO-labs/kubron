import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const lease = defineResource<k8s.LeaseInfo>({
  id: 'lease',
  kind: 'Lease',
  view: 'leases',
  title: 'Leases',
  namespaced: true,
  nav: { group: 'config', label: 'Leases' },
  
  autoRefresh: true,icon: 'configmap',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (l: any) => l.namespace ? { id: 'namespace', ns: '', name: l.namespace } : null },
    { key: 'holder', label: 'Holder' },
    ageCol,
  ],

  properties: [
    { label: 'Holder', value: l => l.holder },
  ],

  api: {
    list: ns => callApp('GetLeases', ns),
    remove: (ns, name) => callApp('DeleteLease', ns, name),
  },
})
