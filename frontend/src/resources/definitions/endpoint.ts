import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const endpoint = defineResource({
  id: 'endpoint',
  kind: 'Endpoint',
  view: 'endpoints',
  title: 'Endpoints',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
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
