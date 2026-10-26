// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, nsCol, ageCol, controlledByProp, managedByProp } from './common'
import BindingSection from '../../components/resource/sections/BindingSection.vue'

export const roleBinding = defineResource({
  id: 'rolebinding',
  kind: 'RoleBinding',
  view: 'rolebindings',
  title: 'Role Bindings',
  // icon: 'bind-role',
  namespaced: true,
  nav: { group: 'access', label: 'Role Bindings' },
  columns: [nameCol, { ...nsCol, link: (r: any) => r.metadata?.namespace ? { id: 'namespace', ns: '', name: r.metadata.namespace } : null }, ageCol],
  properties: [controlledByProp, managedByProp],
  sections: [BindingSection],
  api: {
    list: ns => callApp('GetRoleBindings', ns),
    remove: (ns, n) => callApp('DeleteResource', 'rolebinding', ns, n),
  },
})
