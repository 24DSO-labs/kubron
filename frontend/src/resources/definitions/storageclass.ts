import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const storageClass = defineResource({
  id: 'storageclass',
  kind: 'StorageClass',
  view: 'storageclasses',
  title: 'StorageClasses',
  namespaced: false,

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
    remove: (_ns, name) => callApp('DeleteStorageClass', name),
  },
})
