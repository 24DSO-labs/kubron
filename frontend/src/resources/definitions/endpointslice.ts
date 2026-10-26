// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import EndpointSliceEntriesSection from '../../components/resource/sections/EndpointSliceEntriesSection.vue'

export const endpointSlice = defineResource<k8s.EndpointSliceInfo>({
  id: 'endpointslice',
  kind: 'EndpointSlice',
  view: 'endpointslices',
  title: 'EndpointSlices',
  namespaced: true,
  nav: { group: 'network', label: 'Endpoint Slices' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (e: any) => e.namespace ? { id: 'namespace', ns: '', name: e.namespace } : null },
    { key: 'addressType', label: 'Address Type' },
    { key: 'ports', label: 'Ports', formatter: (e: any) => (e.ports ?? []).join(', ') || '—' },
        { key: 'endpointList', label: 'Endpoints', formatter: (e: any) => e.endpointList || '<none>' },
    ageCol,
  ],

  properties: [
    {
      label: 'Service',
      value: (e: any) => e.service,
      link: (e: any) => (e.service ? { id: 'service', ns: e.namespace, name: e.service } : null),
    },
    { label: 'Address Type', value: (e: any) => e.addressType },
    { label: 'Ports', value: (e: any) => (e.ports ?? []).join(', ') },
    { label: 'Ready endpoints', value: (e: any) => e.endpoints },
    controlledByProp,
    managedByProp,
  ],
  sections: [EndpointSliceEntriesSection],

  api: {
    list: ns => callApp('GetEndpointSlices', ns),
    remove: (ns, name) => callApp('DeleteResource', 'endpointslice', ns, name),
  },
})