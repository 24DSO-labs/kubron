import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import StorageClassParametersTab from '../../components/resource/sections/StorageClassParametersTab.vue'
import StorageClassVolumesTab from '../../components/resource/sections/StorageClassVolumesTab.vue'

const yesNo = (v: boolean) => (v ? 'Yes' : 'No')

export const storageClass = defineResource<k8s.StorageClassInfo>({
  id: 'storageclass',
  kind: 'StorageClass',
  view: 'storageclasses',
  title: 'StorageClasses',
  icon: 'storage-class',
  namespaced: false,
  nav: { group: 'storage', label: 'Storage Classes' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'provisioner', label: 'Provisioner' },
    { key: 'reclaimPolicy', label: 'Reclaim Policy' },
    { key: 'volumeBindingMode', label: 'Binding Mode' },
    {
      key: 'allowVolumeExpansion',
      label: 'Expansion',
      formatter: s => yesNo(s.allowVolumeExpansion),
    },
    {
      key: 'default',
      label: 'Default',
      formatter: s => yesNo(s.default),
      colorFn: s => (s.default ? '#4ade80' : '#868d8a'),
    },
    { key: 'volumes', label: 'PVs', formatter: s => String(s.volumes?.length ?? 0) },
    ageCol,
  ],

  // Name / Age / Created выводит шапка панели; Labels и Annotations — общие секции
  properties: [
    { label: 'Provisioner', value: s => s.provisioner },
    { label: 'Volume Binding Mode', value: s => s.volumeBindingMode },
    { label: 'Reclaim Policy', value: s => s.reclaimPolicy },
    controlledByProp,
    managedByProp,
  ],
  tabs: [
    { id: 'parameters', label: 'Parameters', component: StorageClassParametersTab },
    { id: 'volumes', label: 'Persistent Volumes', component: StorageClassVolumesTab },
  ],

  api: {
    list: () => callApp('GetStorageClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'storageclass', '', name),
  },
})