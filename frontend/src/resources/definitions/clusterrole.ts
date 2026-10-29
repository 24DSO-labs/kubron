// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, ageCol, controlledByProp, managedByProp } from './common'
import RulesSection from '../../components/resource/sections/RulesSection.vue'

export const clusterRole = defineResource({
  id: 'clusterrole',
  kind: 'ClusterRole',
  view: 'clusterroles',
  title: 'Cluster Roles',
  // icon: 'cluster-role',
  namespaced: false,
  nav: { group: 'access', label: 'Cluster Roles' },
  columns: [nameCol, ageCol],
  properties: [{ label: 'defs.columns.rules', value: r => r.rules?.length ?? 0 }, controlledByProp, managedByProp],
  sections: [RulesSection],
  api: {
    list: () => callApp('GetClusterRoles'),
    remove: (_ns, n) => callApp('DeleteResource', 'clusterrole', '', n),
  },
})
