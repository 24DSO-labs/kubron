import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const endpoint = defineResource<k8s.EndpointListInfo>({
  id: 'endpoint',
  kind: 'Endpoint',
  view: 'endpoints',
  title: 'Endpoints',
  namespaced: true,
  nav: { group: 'network', label: 'Endpoints' },
  
  autoRefresh: true,icon: 'endpoint',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (e: any) => e.namespace ? { id: 'namespace', ns: '', name: e.namespace } : null },
    { key: 'addresses', label: 'Addresses' },
    { key: 'ports', label: 'Ports' },
    ageCol,
  ],

  properties: [
    { label: 'Addresses', value: e => e.addresses },
    { label: 'Ports', value: e => e.ports },
  ],

  // Endpoints управляются control plane (Service-контроллером): удалять вручную не нужно
  api: {
    list: ns => callApp('GetEndpoints', ns),
  },
})
