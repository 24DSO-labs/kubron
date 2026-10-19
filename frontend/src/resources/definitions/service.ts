import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { formatMap } from '../utils'
import ServicePortsSection from '../../components/resource/sections/ServicePortsSection.vue'
import EndpointsSection from '../../components/resource/sections/EndpointsSection.vue'

export const service = defineResource<k8s.ServiceInfo, k8s.ServiceDetail>({
  id: 'service',
  kind: 'Service',
  view: 'services',
  title: 'Services',
  namespaced: true,
  nav: { group: 'network', label: 'Services' },
  
  autoRefresh: true,icon: 'service',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (s: any) => s.namespace ? { id: 'namespace', ns: '', name: s.namespace } : null },
    { key: 'clusterIP', label: 'Cluster IP' },
    { key: 'externalIP', label: 'External IP' },
    { key: 'ports', label: 'Ports' },
    ageCol,
  ],

  properties: [
    { label: 'Cluster IP', value: s => s.clusterIP },
    { label: 'Cluster IPs', value: s => s.clusterIPs?.join(', ') },
    { label: 'IP Families', value: s => s.ipFamilies },
    { label: 'IP Family Policy', value: s => s.ipFamilyPolicy },
    { label: 'Type', value: s => s.type },
    { label: 'Selector', value: s => formatMap(s.selector) },
  ],
  sections: [ServicePortsSection, EndpointsSection],

  api: {
    list: ns => callApp('GetServices', ns),
    get: (ns, name) => callApp('GetServiceDetail', ns, name),
    remove: (ns, name) => callApp('DeleteResource', 'service', ns, name),
  },
})
