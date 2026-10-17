import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const ingress = defineResource<k8s.IngressInfo>({
  id: 'ingress',
  kind: 'Ingress',
  view: 'ingresses',
  title: 'Ingresses',
  namespaced: true,
  nav: { group: 'network', label: 'Ingresses' },

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'class', label: 'Class' },
    { key: 'hosts', label: 'Hosts', formatter: i => (i.hosts ?? []).join(', ') },
    { key: 'address', label: 'Address' },
    {
      key: 'tls',
      label: 'TLS',
      formatter: i => (i.tls ? 'True' : 'False'),
    },
    ageCol,
  ],

  properties: [
    { label: 'Class', value: i => i.class },
    { label: 'Hosts', value: i => (i.hosts ?? []).join(', ') },
    { label: 'Address', value: i => i.address },
    { label: 'Paths', value: i => i.paths },
    { label: 'TLS', value: i => (i.tls ? 'True' : 'False') },
  ],

  api: {
    list: ns => callApp('GetIngresses', ns),
    remove: (ns, name) => callApp('DeleteIngress', ns, name),
  },
})
