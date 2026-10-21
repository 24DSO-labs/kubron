import { defineResource } from '../types'
import { Perm } from '../../types/auth'
import type { k8s as models } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { useK8s } from '../../composables/useK8s'
import { flatNameCol, flatNsCol, ageCol, managedByProp } from './common'
import { badgeClass, statusColor } from '../status'
import { containersOf } from '../utils'

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

/** Маппинг kind -> resource id для ссылок */
const kindToId: Record<string, string> = {
  ReplicaSet: 'replicaset',
  StatefulSet: 'statefulset',
  DaemonSet: 'daemonset',
  Job: 'job',
  CronJob: 'cronjob',
  Deployment: 'deployment',
  Node: 'node',
  Service: 'service',
}

function controlledByLink(p: any): { id: string; ns: string; name: string } | null {
  const cb = p.controlledBy
  if (!cb) return null
  const [kind, name] = cb.split('/')
  if (!kind || !name) return null
  const id = kindToId[kind]
  if (!id) return null
  return { id, ns: p.namespace, name }
}

export const pod = defineResource<models.PodInfo, models.PodDetail>({
  id: 'pod',
  kind: 'Pod',
  view: 'pods',
  title: 'Pods',
  namespaced: true,
  nav: { group: 'workloads', label: 'Pods' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (p: any) => p.namespace ? { id: 'namespace', ns: '', name: p.namespace } : null },
    {
  key: 'cpu', label: 'CPU', className: 'mono metric-cell',
  animated: true, width: 100,
  formatter: (p: any) => k8s.formatCpu(p.cpu),
},
{
  key: 'memory', label: 'MEMORY', className: 'mono metric-cell',
  animated: true, width: 110,
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
    { key: 'node', label: 'NODE', link: (p: any) => p.node ? { id: 'node', ns: '', name: p.node } : null },
    {
      key: 'controlledBy',
      label: 'Controller',
      className: 'ns-label',
      formatter: (p: any) => p.controlledBy?.split('/')[0] || '—',
      link: controlledByLink,
    },
    
    { key: 'qos', label: 'QOS' },
  ],

  headSections: [PodMetricsSection],
  properties: [
    { label: 'Status', value: p => podStatus(p), badge: p => badgeClass(podStatus(p)) },
    { label: 'Ready', value: p => p.ready },
    { label: 'Restarts', value: p => p.restarts },
    { label: 'Node', value: p => p.node, link: p => p.node ? { id: 'node', ns: '', name: p.node } : null },
    { label: 'IP', value: p => p.ip },
    {
      label: 'Controlled By',
      value: p => {
        const cb = p.controlledBy
        if (!cb) return null
        const [kind] = cb.split('/')
        const hasLink = kind && kindToId[kind]
        return hasLink ? kind + ' ' : cb
      },
      linkLabel: p => {
        const cb = p.controlledBy
        if (!cb) return null
        const [, name] = cb.split('/')
        return name || null
      },
      link: controlledByLink,
    },
    { label: 'Service Account', value: p => p.serviceAccount, link: p => p.serviceAccount ? { id: 'serviceaccount', ns: p.namespace, name: p.serviceAccount } : null },
    { label: 'QoS', value: p => p.qos },
    managedByProp,
  ],
  sections: [PodVolumesSection, TolerationsSection, ConditionsSection],
  tabs: [
    { id: 'containers', label: 'Containers', component: ContainersTab },
  ],

  actions: [
    {
      id: 'attach',
      permission: Perm.Exec,
      label: 'Attach',
      icon: 'attach',
      perContainer: true,
      run: (p, ctx, container) => {
        const c = container ?? containersOf(p)[0]?.name
        if (!c) return ctx.notifyError('No containers available')
        return ctx.openAttach(p.namespace, p.name, c)
      },
    },
    {
      id: 'shell',
      permission: Perm.Exec,
      label: 'Shell',
      icon: 'terminal',
      perContainer: true,
      run: (p, ctx, container) => {
        const c = container ?? containersOf(p)[0]?.name
        if (!c) return ctx.notifyError('No containers available')
        return ctx.openShell(p.namespace, p.name, c)
      },
    },
    {
      id: 'logs',
      permission: Perm.Read,
      label: 'Logs',
      icon: 'logs',
      perContainer: true,
      run: (p, ctx, container) => {
        const containers = containersOf(p)
        ctx.openLogs(p.namespace, p.name, containers, container ?? containers[0]?.name)
      },
    },
    {
      id: 'evict',
      label: 'Evict',
      icon: 'evict',
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
    remove: (ns, name) => callApp('DeleteResource', 'pod', ns, name),
  },
})
