import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const persistentVolume = defineResource<k8s.PVInfo>({
  id: 'persistentvolume',
  kind: 'PersistentVolume',
  view: 'pv',
  title: 'PersistentVolumes',
  namespaced: false,
  nav: { group: 'storage', label: 'Persistent Volumes' },
  
  autoRefresh: true,icon: 'pvc',

  columns: [
    flatNameCol,
    { key: 'capacity', label: 'Capacity' },
    { key: 'accessModes', label: 'Access Modes' },
    { key: 'reclaimPolicy', label: 'Reclaim Policy' },
    {
      key: 'status',
      label: 'Status',
      colorFn: pv => statusColor(pv.status),
    },
    { key: 'claim', label: 'Claim' },
    ageCol,
  ],

  properties: [
    { label: 'Status', value: pv => pv.status, badge: pv => badgeClass(pv.status) },
    { label: 'Capacity', value: pv => pv.capacity },
    { label: 'Access Modes', value: pv => pv.accessModes },
    { label: 'Reclaim Policy', value: pv => pv.reclaimPolicy },
    { label: 'Claim', value: pv => pv.claim },
    { label: 'Storage Class', value: pv => pv.storageClass },
  ],

  api: {
    list: () => callApp('GetPersistentVolumes'),
    remove: (_ns, name) => callApp('DeletePersistentVolume', name),
  },
})
