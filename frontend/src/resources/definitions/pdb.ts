import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const podDisruptionBudget = defineResource({
  id: 'poddisruptionbudget',
  kind: 'PDB',
  view: 'pdb',
  title: 'PodDisruptionBudgets',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'minAvailable', label: 'Min Available' },
    { key: 'maxUnavailable', label: 'Max Unavailable' },
    { key: 'currentHealthy', label: 'Current Healthy' },
    { key: 'desiredHealthy', label: 'Desired Healthy' },
    { key: 'disruptionsAllowed', label: 'Allowed Disruptions' },
    ageCol,
  ],

  properties: [
    { label: 'Min Available', value: p => p.minAvailable },
    { label: 'Max Unavailable', value: p => p.maxUnavailable },
    { label: 'Current Healthy', value: p => p.currentHealthy },
    { label: 'Desired Healthy', value: p => p.desiredHealthy },
    { label: 'Allowed Disruptions', value: p => p.disruptionsAllowed },
  ],

  api: {
    list: ns => callApp('GetPodDisruptionBudgets', ns),
    remove: (ns, name) => callApp('DeletePDB', ns, name),
  },
})
