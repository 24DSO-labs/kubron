import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const horizontalPodAutoscaler = defineResource({
  id: 'horizontalpodautoscaler',
  kind: 'HPA',
  view: 'hpa',
  title: 'HorizontalPodAutoscalers',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'reference', label: 'Reference' },
    { key: 'minReplicas', label: 'Min' },
    { key: 'maxReplicas', label: 'Max' },
    { key: 'currentReplicas', label: 'Current' },
    ageCol,
  ],

  properties: [
    { label: 'Reference', value: h => h.reference },
    { label: 'Min Replicas', value: h => h.minReplicas },
    { label: 'Max Replicas', value: h => h.maxReplicas },
    { label: 'Current Replicas', value: h => h.currentReplicas },
    { label: 'Desired Replicas', value: h => h.desiredReplicas },
  ],

  api: {
    list: ns => callApp('GetHPA', ns),
    remove: (ns, name) => callApp('DeleteHPA', ns, name),
  },
})
