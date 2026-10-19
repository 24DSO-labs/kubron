import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const networkPolicy = defineResource<k8s.NetworkPolicyInfo>({
  id: 'networkpolicy',
  kind: 'NetworkPolicy',
  view: 'network-policies',
  title: 'NetworkPolicies',
  namespaced: true,
  nav: { group: 'network', label: 'Network Policies' },
  
  autoRefresh: true,
  // icon: 'network-policy',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (np: any) => np.namespace ? { id: 'namespace', ns: '', name: np.namespace } : null },
    { key: 'podSelector', label: 'Pod Selector' },
    ageCol,
  ],

  properties: [
    { label: 'Pod Selector', value: np => np.podSelector },
  ],

  api: {
    list: ns => callApp('GetNetworkPolicies', ns),
    remove: (ns, name) => callApp('DeleteResource', 'networkpolicy', ns, name),
  },
})
