import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const storageClass = defineResource<k8s.StorageClassInfo>({
  id: 'storageclass',
  kind: 'StorageClass',
  view: 'storageclasses',
  title: 'StorageClasses',
  namespaced: false,
  nav: { group: 'storage', label: 'Storage Classes' },
  
  autoRefresh: true,icon: 'storage-class',

  columns: [
    flatNameCol,
    { key: 'provisioner', label: 'Provisioner' },
    { key: 'reclaimPolicy', label: 'Reclaim Policy' },
    {
      key: 'default',
      label: 'Default',
      formatter: s => (s.default ? 'True' : 'False'),
    },
    ageCol,
  ],

  properties: [
    { label: 'Provisioner', value: s => s.provisioner },
    { label: 'Reclaim Policy', value: s => s.reclaimPolicy },
    { label: 'Default', value: s => (s.default ? 'True' : 'False') },
  ],

  api: {
    list: () => callApp('GetStorageClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'storageclass', '', name),
  },
})
