import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'
import NodeMetricsSection from '../../components/resource/sections/NodeMetricsSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import NodeTaintsSection from '../../components/resource/sections/NodeTaintsSection.vue'

export const node = defineResource({
  id: 'node',
  kind: 'Node',
  view: 'nodes',
  title: 'Nodes',
  namespaced: false,
  nav: { label: 'Nodes' },

  columns: [
    flatNameCol,
    { key: 'status', label: 'Status', colorFn: n => statusColor(n.status) },
    { key: 'roles', label: 'Roles' },
    { key: 'version', label: 'Version' },
    { key: 'cpuCapacity', label: 'CPU' },
    { key: 'memoryCapacity', label: 'Memory' },
    ageCol,
  ],

  properties: [
    { label: 'Status', value: n => n.status, badge: n => badgeClass(n.status) },
    { label: 'Role', value: n => n.roles },
    { label: 'Kubernetes Version', value: n => n.kubeVersion },
    { label: 'Container Runtime', value: n => n.containerRuntime },
    { label: 'OS Image', value: n => n.osImage },
    { label: 'Architecture', value: n => n.architecture },
    { label: 'Internal IP', value: n => n.internalIP },
    { label: 'External IP', value: n => n.externalIP },
    { label: 'Pod CIDR', value: n => n.podCIDR },
    { label: 'Allocatable CPU', value: n => n.allocatableCPU },
    { label: 'Allocatable Memory', value: n => n.allocatableMemory },
    { label: 'Allocatable Pods', value: n => n.allocatablePods },
  ],
  sections: [NodeMetricsSection, ConditionsSection, NodeTaintsSection],

  api: {
    list: () => callApp('GetNodes'),
    get: (_ns, name) => callApp('GetNodeDetail', name),
  },
})