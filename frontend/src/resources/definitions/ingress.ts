import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import IngressRulesSection from '../../components/resource/sections/IngressRulesSection.vue'

const classLink = (i: any) => (i.class ? { id: 'ingressclass', ns: '', name: i.class } : null)

export const ingress = defineResource<k8s.IngressInfo>({
  id: 'ingress',
  kind: 'Ingress',
  view: 'ingresses',
  title: 'Ingresses',
  namespaced: true,
  nav: { group: 'network', label: 'Ingresses' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (i: any) => i.namespace ? { id: 'namespace', ns: '', name: i.namespace } : null },
    { key: 'class', label: 'Class', link: classLink },
    { key: 'address', label: 'Load Balancers', formatter: (i: any) => i.address || '—' },
    { key: 'hosts', label: 'Hosts', formatter: (i: any) => [...new Set(i.hosts ?? [])].join(', ') || '—' },
    { key: 'paths', label: 'Rules', formatter: (i: any) => i.paths || '—' },
    { key: 'tls', label: 'TLS', formatter: (i: any) => (i.tls ? 'True' : 'False') },
    ageCol,
  ],

  properties: [
    { label: 'Ingress Class', value: (i: any) => i.class, link: classLink },
    { label: 'Load Balancers', value: (i: any) => i.address },
    { label: 'Hosts', value: (i: any) => [...new Set(i.hosts ?? [])].join(', ') },
    { label: 'TLS', value: (i: any) => (i.tls ? 'True' : 'False') },
  ],
  sections: [IngressRulesSection],

  api: {
    list: ns => callApp('GetIngresses', ns),
    remove: (ns, name) => callApp('DeleteResource', 'ingress', ns, name),
  },
})