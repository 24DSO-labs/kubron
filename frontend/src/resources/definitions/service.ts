// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import { formatMap } from '../utils'
import { statusColor } from '../status'
import ConnectionTab from '../../components/resource/sections/ConnectionTab.vue'
import EndpointSlicesTab from '../../components/resource/sections/EndpointSlicesTab.vue'
import EndpointsTab from '../../components/resource/sections/EndpointsTab.vue'
import ServiceNetworkPoliciesTab from '../../components/resource/sections/ServiceNetworkPoliciesTab.vue'

export const service = defineResource<k8s.ServiceInfo, k8s.ServiceDetail>({
  id: 'service',
  kind: 'Service',
  view: 'services',
  title: 'Services',
  namespaced: true,
  nav: { group: 'network', label: 'Services' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (s: any) => s.namespace ? { id: 'namespace', ns: '', name: s.namespace } : null },
    { key: 'type', label: 'Type' },
    { key: 'clusterIP', label: 'Cluster IP' },
    { key: 'ports', label: 'Ports' },
    { key: 'externalIP', label: 'External IP' },
    { key: 'selector', label: 'Selector', formatter: (s: any) => formatMap(s.selector) },
    ageCol,
    { key: 'status', label: 'Status', colorFn: (s: any) => statusColor(s.status) },
  ],

  properties: [
    { label: 'Session Affinity', value: s => s.sessionAffinity },
    { label: 'Type', value: s => s.type },
    { label: 'Selector', value: s => formatMap(s.selector) },
    { label: 'Finalizers', value: s => s.finalizers?.join(', ') },
    { label: 'External Name', value: (s: any) => s.externalName },
    { label: 'Load Balancer', value: (s: any) => (s.loadBalancer ?? []).join(', ') },
    { label: 'External Traffic Policy', value: (s: any) => s.externalTrafficPolicy },
    { label: 'Internal Traffic Policy', value: (s: any) => s.internalTrafficPolicy },
    
    controlledByProp,
    managedByProp,
  ],
  sections: [],
  tabs: [
    { id: 'connection', label: 'Connection', component: ConnectionTab },
    { id: 'endpoints', label: 'Endpoints', component: EndpointsTab },
    { id: 'endpointSlices', label: 'Endpoint Slices', component: EndpointSlicesTab },
    { id: 'networkPolicies', label: 'Network Policies', component: ServiceNetworkPoliciesTab },
  ],

  api: {
    list: ns => callApp('GetServices', ns),
    get: (ns, name) => callApp('GetServiceDetail', ns, name),
    remove: (ns, name) => callApp('DeleteResource', 'service', ns, name),
  },
})
