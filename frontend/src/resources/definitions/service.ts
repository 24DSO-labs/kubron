import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import ServicePortsSection from '../../components/resource/sections/ServicePortsSection.vue'
import EndpointsSection from '../../components/resource/sections/EndpointsSection.vue'

export const service = defineResource({
  id: 'service',
  kind: 'Service',
  view: 'services',
  title: 'Services',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'type', label: 'Type' },
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
    { label: 'Selector', value: s => s.selector },
  ],
  sections: [ServicePortsSection, EndpointsSection],

  api: {
    list: ns => callApp('GetServices', ns),
    get: (ns, name) => callApp('GetServiceDetail', ns, name),
    remove: (ns, name) => callApp('DeleteService', ns, name),
  },
})
