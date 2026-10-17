import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const lease = defineResource({
  id: 'lease',
  kind: 'Lease',
  view: 'leases',
  title: 'Leases',
  namespaced: true,
  nav: { group: 'config', label: 'Leases' },

  columns: [
    flatNameCol,
    flatNsCol,
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
