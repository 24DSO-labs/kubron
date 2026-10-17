import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

export const replicaSet = defineResource({
  id: 'replicaset',
  kind: 'ReplicaSet',
  view: 'replicasets',
  title: 'ReplicaSets',
  namespaced: true,
  nav: { group: 'workloads', label: 'Replica Sets' },

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'desired', label: 'Desired' },
    { key: 'ready', label: 'Ready' },
    ageCol,
  ],

  properties: [
    { label: 'Desired', value: r => r.desired },
    { label: 'Ready', value: r => r.ready },
  ],
  sections: [ImagesSection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  api: {
    list: ns => callApp('GetReplicaSets', ns),
    remove: (ns, n) => callApp('DeleteReplicaSet', ns, n),
  },
})
