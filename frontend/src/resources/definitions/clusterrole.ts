import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, ageCol } from './common'
import RulesSection from '../../components/resource/sections/RulesSection.vue'

export const clusterRole = defineResource({
  id: 'clusterrole',
  kind: 'ClusterRole',
  view: 'clusterroles',
  title: 'Cluster Roles',
  icon: '🔑',
  namespaced: false,
  columns: [nameCol, ageCol],
  properties: [{ label: 'Rules', value: r => r.rules?.length ?? 0 }],
  sections: [RulesSection],
  api: {
    list: () => callApp('GetClusterRoles'),
    remove: (_ns, n) => callApp('DeleteClusterRole', n),
  },
})
