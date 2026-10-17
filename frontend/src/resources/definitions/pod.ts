import { defineResource } from '../types'
import type { k8s as models } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { useK8s } from '../../composables/useK8s'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'
import attachIcon from '../../assets/icons/attach.svg'
import terminalIcon from '../../assets/icons/terminal.svg'
import logsIcon from '../../assets/icons/logs.svg'
import evictIcon from '../../assets/icons/evict.svg'
import PodContainersCell from '../../components/resource/sections/PodContainersCell.vue'
import PodMetricsSection from '../../components/resource/sections/PodMetricsSection.vue'
import PodVolumesSection from '../../components/resource/sections/PodVolumesSection.vue'
import ContainersTab from '../../components/resource/sections/ContainersTab.vue'
import TolerationsSection from '../../components/resource/sections/TolerationsSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'

const k8s = useK8s()

function podStatus(p: any): string {
  return p.status || 'Unknown'
}

export const pod = defineResource<models.PodInfo, models.PodDetail>({
  id: 'pod',
  kind: 'Pod',
  view: 'pods',
  title: 'Pods',
  namespaced: true,
  nav: { group: 'workloads', label: 'Pods' },

  columns: [
    flatNameCol,
    flatNsCol,
    {
      key: 'cpu',
      label: 'CPU',
      className: 'mono metric-cell',
      formatter: (p: any) => k8s.formatCpu(p.cpu),
    },
    {
      key: 'memory',
      label: 'MEMORY',
      className: 'mono metric-cell',
      formatter: (p: any) => k8s.formatMemory(p.memory),
    },
    {
      key: 'containers',
      label: 'CONTAINERS',
      component: PodContainersCell,
    },
    {
      key: 'status',
      label: 'STATUS',
      className: 'status-label',
      formatter: podStatus,
      colorFn: p => statusColor(podStatus(p)),
    },
    { key: 'restarts', label: 'RESTARTS' },
    ageCol,
    { key: 'node', label: 'NODE' },
    { key: 'qos', label: 'QOS' },
  ],

  headSections: [PodMetricsSection],
  properties: [
    { label: 'Status', value: p => podStatus(p), badge: p => badgeClass(podStatus(p)) },
    { label: 'Ready', value: p => p.ready },
    { label: 'Restarts', value: p => p.restarts },
    { label: 'Node', value: p => p.node },
    { label: 'IP', value: p => p.ip },
    { label: 'Controlled By', value: p => p.controlledBy },
    { label: 'Service Account', value: p => p.serviceAccount },
    { label: 'QoS', value: p => p.qos },
  ],
  sections: [PodVolumesSection, TolerationsSection, ConditionsSection],
  tabs: [
    { id: 'containers', label: 'Containers', component: ContainersTab },
  ],

  actions: [
    {
      id: 'attach',
      label: 'Attach',
      icon: attachIcon,
      perContainer: true,
      run: (p, ctx, container) => {
        const c = container ?? p.containers?.[0]?.name
        if (!c) return ctx.notifyError('No containers available')
        return ctx.openAttach(p.namespace, p.name, c)
      },
    },
    {
      id: 'shell',
      label: 'Shell',
      icon: terminalIcon,
      perContainer: true,
      run: (p, ctx, container) => {
        const c = container ?? p.containers?.[0]?.name
        if (!c) return ctx.notifyError('No containers available')
        return ctx.openShell(p.namespace, p.name, c)
      },
    },
    {
      id: 'logs',
      label: 'Logs',
      icon: logsIcon,
      perContainer: true,
      run: (p, ctx, container) => {
        const containers = p.containers?.filter((c: any) => !c.isInit) ?? []
        ctx.openLogs(p.namespace, p.name, containers, container ?? containers[0]?.name)
      },
    },
    {
      id: 'evict',
      label: 'Evict',
      icon: evictIcon,
      danger: true,
      run: async (p, ctx) => {
        if (!(await ctx.confirm(`Evict pod ${p.name}?`))) return
        return ctx.perform(
          () => callApp('EvictPod', p.namespace, p.name),
          `Pod ${p.name} evicted`,
        )
      },
    },
  ],

  // Список показывается сразу, метрики подмешиваются, когда придут
  onLoaded: async (items) => {
    await k8s.loadPodMetrics()
    return k8s.mergePodMetrics(items, k8s.podMetrics.value)
  },

  api: {
    list: ns => callApp('GetPods', ns),
    get: (ns, name) => callApp('GetPodDetail', ns, name),
    remove: (ns, name) => callApp('DeletePod', ns, name),
  },
})
