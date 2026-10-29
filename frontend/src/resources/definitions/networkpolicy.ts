// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import NetworkPolicyRulesSection from '../../components/resource/sections/NetworkPolicyRulesSection.vue'

export const networkPolicy = defineResource<k8s.NetworkPolicyInfo>({
  id: 'networkpolicy',
  kind: 'NetworkPolicy',
  view: 'network-policies',
  title: 'NetworkPolicies',
  namespaced: true,
  nav: { group: 'network', label: 'Network Policies' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (np: any) => np.namespace ? { id: 'namespace', ns: '', name: np.namespace } : null },
    { key: 'podSelector', label: 'defs.columns.podSelector' },
    { key: 'policyTypes', label: 'defs.columns.policyTypes', formatter: (np: any) => (np.policyTypes ?? []).join(', ') },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.podSelector', value: (np: any) => np.podSelector },
    { label: 'defs.columns.policyTypes', value: (np: any) => (np.policyTypes ?? []).join(', ') },
    { label: 'defs.props.ingressRules', value: (np: any) => (np.policyTypes ?? []).includes('Ingress') ? (np.ingress?.length ?? 0) : '—' },
    { label: 'defs.props.egressRules', value: (np: any) => (np.policyTypes ?? []).includes('Egress') ? (np.egress?.length ?? 0) : '—' },
    controlledByProp,
    managedByProp,
  ],
  sections: [NetworkPolicyRulesSection],

  api: {
    list: ns => callApp('GetNetworkPolicies', ns),
    remove: (ns, name) => callApp('DeleteResource', 'networkpolicy', ns, name),
  },
})