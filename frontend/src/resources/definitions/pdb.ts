// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import PVCSelectorSection from '../../components/resource/sections/PVCSelectorSection.vue'

export const podDisruptionBudget = defineResource<k8s.PDBInfo>({
  id: 'poddisruptionbudget',
  kind: 'PDB',
  view: 'pdb',
  title: 'PodDisruptionBudgets',
  namespaced: true,
  nav: { group: 'config', label: 'Pod Disruption Budgets' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (p: any) => p.namespace ? { id: 'namespace', ns: '', name: p.namespace } : null },
    { key: 'minAvailable', label: 'defs.columns.minAvailable' },
    { key: 'maxUnavailable', label: 'defs.columns.maxUnavailable' },
    { key: 'currentHealthy', label: 'defs.columns.currentHealthy' },
    { key: 'desiredHealthy', label: 'defs.columns.desiredHealthy' },
    { key: 'disruptionsAllowed', label: 'defs.columns.allowedDisruptions' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.minAvailable', value: (p: any) => p.minAvailable },
    { label: 'defs.columns.maxUnavailable', value: (p: any) => p.maxUnavailable },
    { label: 'defs.columns.currentHealthy', value: (p: any) => p.currentHealthy },
    { label: 'defs.columns.desiredHealthy', value: (p: any) => p.desiredHealthy },
    { label: 'defs.columns.allowedDisruptions', value: (p: any) => p.disruptionsAllowed },
    { label: 'defs.props.expectedPods', value: (p: any) => p.expectedPods },
    { label: 'defs.props.unhealthyPodEviction', value: (p: any) => p.unhealthyPodEvictionPolicy },
    { label: 'defs.props.finalizers', value: (p: any) => (p.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],
  sections: [ConditionsSection, PVCSelectorSection],

  api: {
    list: ns => callApp('GetPodDisruptionBudgets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'poddisruptionbudget', ns, name),
  },
})
