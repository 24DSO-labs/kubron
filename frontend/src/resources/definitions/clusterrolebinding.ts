import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, ageCol } from './common'
import BindingSection from '../../components/resource/sections/BindingSection.vue'

export const clusterRoleBinding = defineResource({
  id: 'clusterrolebinding',
  kind: 'ClusterRoleBinding',
  view: 'clusterrolebindings',
  title: 'Cluster Role Bindings',
  icon: '🔗',
  namespaced: false,
  columns: [nameCol, ageCol],
  sections: [BindingSection],
  api: {
    list: () => callApp('GetClusterRoleBindings'),
    remove: (_ns, n) => callApp('DeleteClusterRoleBinding', n),
  },
})
