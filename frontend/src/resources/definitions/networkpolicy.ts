import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const networkPolicy = defineResource({
  id: 'networkpolicy',
  kind: 'NetworkPolicy',
  view: 'network-policies',
  title: 'NetworkPolicies',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'podSelector', label: 'Pod Selector' },
    ageCol,
  ],

  properties: [
    { label: 'Pod Selector', value: np => np.podSelector },
  ],

  api: {
    list: ns => callApp('GetNetworkPolicies', ns),
    remove: (ns, name) => callApp('DeleteNetworkPolicy', ns, name),
  },
})
