import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const persistentVolumeClaim = defineResource<k8s.PVCInfo>({
  id: 'persistentvolumeclaim',
  kind: 'PersistentVolumeClaim',
  view: 'pvc',
  title: 'PersistentVolumeClaims',
  namespaced: true,
  nav: { group: 'storage', label: 'Persistent Volume Claims' },

  columns: [
    flatNameCol,
    flatNsCol,
    {
      key: 'status',
      label: 'Status',
      colorFn: p => statusColor(p.status),
    },
    { key: 'volume', label: 'Volume' },
    { key: 'capacity', label: 'Capacity' },
    { key: 'storageClass', label: 'Storage Class' },
    ageCol,
  ],

  properties: [
    { label: 'Status', value: p => p.status, badge: p => badgeClass(p.status) },
    { label: 'Volume', value: p => p.volume },
    { label: 'Capacity', value: p => p.capacity },
    { label: 'Access Modes', value: p => p.accessModes },
    { label: 'Storage Class', value: p => p.storageClass },
  ],

  api: {
    list: ns => callApp('GetPersistentVolumeClaims', ns),
    remove: (ns, name) => callApp('DeletePersistentVolumeClaim', ns, name),
  },
})
