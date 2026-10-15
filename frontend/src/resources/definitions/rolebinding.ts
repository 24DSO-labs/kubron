import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, nsCol, ageCol } from './common'
import BindingSection from '../../components/resource/sections/BindingSection.vue'

export const roleBinding = defineResource({
  id: 'rolebinding',
  kind: 'RoleBinding',
  view: 'rolebindings',
  title: 'Role Bindings',
  icon: '🔗',
  namespaced: true,
  columns: [nameCol, nsCol, ageCol],
  sections: [BindingSection],
  api: {
    list: ns => callApp('GetRoleBindings', ns),
    remove: (ns, n) => callApp('DeleteRoleBinding', ns, n),
  },
})
