import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, nsCol, ageCol, controlledByProp, managedByProp } from './common'
import RulesSection from '../../components/resource/sections/RulesSection.vue'

export const role = defineResource({
  id: 'role',
  kind: 'Role',
  view: 'roles',
  title: 'Roles',
  // icon: 'cluster-role',
  namespaced: true,
  nav: { group: 'access', label: 'Roles' },
  columns: [nameCol, { ...nsCol, link: (r: any) => r.metadata?.namespace ? { id: 'namespace', ns: '', name: r.metadata.namespace } : null }, ageCol],
  properties: [{ label: 'Rules', value: r => r.rules?.length ?? 0 }, controlledByProp, managedByProp],
  sections: [RulesSection],
  api: {
    list: ns => callApp('GetRoles', ns),
    remove: (ns, n) => callApp('DeleteResource', 'role', ns, n),
  },
})
