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
    { key: 'minAvailable', label: 'Min Available' },
    { key: 'maxUnavailable', label: 'Max Unavailable' },
    { key: 'currentHealthy', label: 'Current Healthy' },
    { key: 'desiredHealthy', label: 'Desired Healthy' },
    { key: 'disruptionsAllowed', label: 'Allowed Disruptions' },
    ageCol,
  ],

  properties: [
    { label: 'Min Available', value: (p: any) => p.minAvailable },
    { label: 'Max Unavailable', value: (p: any) => p.maxUnavailable },
    { label: 'Current Healthy', value: (p: any) => p.currentHealthy },
    { label: 'Desired Healthy', value: (p: any) => p.desiredHealthy },
    { label: 'Allowed Disruptions', value: (p: any) => p.disruptionsAllowed },
    { label: 'Expected Pods', value: (p: any) => p.expectedPods },
    { label: 'Unhealthy Pod Eviction', value: (p: any) => p.unhealthyPodEvictionPolicy },
    { label: 'Finalizers', value: (p: any) => (p.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],
  sections: [ConditionsSection, PVCSelectorSection],

  api: {
    list: ns => callApp('GetPodDisruptionBudgets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'poddisruptionbudget', ns, name),
  },
})
