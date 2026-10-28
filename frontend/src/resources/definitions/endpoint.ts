// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import EndpointSubsetsSection from '../../components/resource/sections/EndpointSubsetsSection.vue'

export const endpoint = defineResource<k8s.EndpointListInfo>({
  id: 'endpoint',
  kind: 'Endpoints',
  view: 'endpoints',
  title: 'Endpoints',
  namespaced: true,
  nav: { group: 'network', label: 'Endpoints' },
  autoRefresh: true,
  // Endpoints создаёт control plane по Service: удаление не предлагаем
  columns: [
    flatNameCol,
    { ...flatNsCol, link: (e: any) => e.namespace ? { id: 'namespace', ns: '', name: e.namespace } : null },
    { key: 'endpoints', label: 'defs.columns.endpoints' },
    ageCol,
  ],
  properties: [
    { label: 'defs.props.service', value: e => e.name, link: e => ({ id: 'service', ns: e.namespace, name: e.name }) },
    { label: 'defs.props.readyAddresses', value: e => e.addresses },
    controlledByProp,
    managedByProp,
  ],
  sections: [EndpointSubsetsSection],
  api: { list: ns => callApp('GetEndpoints', ns) },
})