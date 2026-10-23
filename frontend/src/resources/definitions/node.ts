import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import NodeUsageCell from '../../components/resource/sections/NodeUsageCell.vue'
import NodeMetricsSection from '../../components/resource/sections/NodeMetricsSection.vue'
import NodeTaintsSection from '../../components/resource/sections/NodeTaintsSection.vue'
import NodeConditionsSection from '../../components/resource/sections/NodeConditionsSection.vue'
import NodeCapacityTab from '../../components/resource/sections/NodeCapacityTab.vue'
import NodeAllocatableTab from '../../components/resource/sections/NodeAllocatableTab.vue'
import NodePodsTab from '../../components/resource/sections/NodePodsTab.vue'
import { Perm } from '../../types/auth'

/** Цвет Conditions: зелёный — Ready без проблем, жёлтый — Ready с pressure, красный — NotReady */
const conditionsColor = (n: any): string => {
  if (n.status !== 'Ready') return '#ef4444'
  return n.conditions === 'Ready' ? '#4ade80' : '#facc15'
}

export const node = defineResource<k8s.NodeInfo, k8s.NodeDetail>({
  id: 'node',
  kind: 'Node',
  view: 'nodes',
  title: 'Nodes',
  namespaced: false,
  nav: { label: 'Nodes' },

  autoRefresh: true,
  icon: 'service',

  columns: [
    flatNameCol,
    { key: 'cpu', label: 'CPU', width: 100, component: NodeUsageCell },
    { key: 'memory', label: 'Memory', width: 100, component: NodeUsageCell },
    { key: 'disk', label: 'Disk', width: 100, component: NodeUsageCell },
    { key: 'taints', label: 'Taints', width: 80 },
    // kubectl показывает <none> для worker, в списке оставляем пусто
    { key: 'roles', label: 'Roles', formatter: n => (n.roles === '<none>' ? '' : n.roles) },
    { key: 'version', label: 'Version' },
    ageCol,
    { key: 'conditions', label: 'Conditions', colorFn: conditionsColor },
  ],

  // Использование CPU / памяти / диска подмешивается к строкам, когда придут метрики
  onLoaded: async items => {
    let list: any[] = []
    try {
      list = ((await callApp('GetNodeMetrics')) as any[]) ?? []
    } catch {
      /* стек метрик не установлен или Prometheus недоступен: список остаётся без полосок */
    }
    if (!list.length) return items
    const byName = new Map(list.map(m => [m.name, m]))
    return items.map(n => {
      const m = byName.get(n.name)
      if (!m) return n
      return {
        ...n,
        usage: {
          cpu: { pct: m.cpuPercent, used: m.cpuUsedCores, total: m.cpuCores },
          memory: { pct: m.memoryPercent, used: m.memoryUsedBytes, total: m.memoryTotalBytes },
          disk: { pct: m.diskPercent, used: m.diskUsedBytes, total: m.diskTotalBytes },
        },
      }
    }) as any[]
  },

  // Панель: Metrics -> Labels -> Annotations -> properties -> sections (Taints, Conditions)
  headSections: [NodeMetricsSection],
  properties: [
    {
      label: 'Addresses',
      multiline: true,
      value: n => (n.addresses ?? []).map(a => `${a.type}: ${a.address}`).join('\n'),
    },
    { label: 'OS', value: n => (n.operatingSystem ? `${n.operatingSystem} (${n.architecture})` : n.architecture) },
    { label: 'OS Image', value: n => n.osImage },
    { label: 'Kernel version', value: n => n.kernelVersion },
    { label: 'Container runtime', value: n => n.containerRuntime },
    { label: 'Kubelet version', value: n => n.kubeVersion },
    controlledByProp,
    managedByProp,
  ],
  sections: [NodeTaintsSection, NodeConditionsSection],
  tabs: [
    { id: 'capacity', label: 'Capacity', component: NodeCapacityTab },
    { id: 'allocatable', label: 'Allocatable', component: NodeAllocatableTab },
    { id: 'pods', label: 'Pods', component: NodePodsTab },
  ],

  actions: [
    {
      id: 'shell',
      label: 'Shell',
      icon: 'terminal',
      permission: Perm.Exec,
      run: async (n, ctx) => {
        if (!(await ctx.confirm(
          `Open a root shell on node ${n.name}? A privileged helper pod is created in kube-system and removed when the session ends.`,
        ))) return
        try {
          const s = await callApp('StartNodeShell', n.name)
          await ctx.openAttach(s.namespace, s.pod, s.container)
        } catch (e: any) {
          ctx.notifyError(`Shell: ${e?.message || e}`)
        }
      },
    },
    {
      id: 'cordon',
      label: 'Cordon',
      icon: 'pause',
      permission: Perm.Operate,
      when: n => !n.unschedulable,
      run: (n, ctx) =>
        ctx.perform(() => callApp('CordonNode', n.name, true), `Node ${n.name} cordoned`),
    },
    {
      id: 'uncordon',
      label: 'Uncordon',
      icon: 'play',
      permission: Perm.Operate,
      when: n => !!n.unschedulable,
      run: (n, ctx) =>
        ctx.perform(() => callApp('CordonNode', n.name, false), `Node ${n.name} uncordoned`),
    },
    {
      id: 'drain',
      label: 'Drain',
      icon: 'evict',
      permission: Perm.Write,
      run: async (n, ctx) => {
        if (!(await ctx.confirm(
          `Drain node ${n.name}? It will be cordoned and its pods evicted (DaemonSet and static pods are skipped). This can take a few minutes.`,
        ))) return
        return ctx.perform(() => callApp('DrainNode', n.name), `Node ${n.name} drained`)
      },
    },
  ],

  // Узлы удаляют осознанно и по одному
  bulkDelete: false,

  api: {
    list: () => callApp('GetNodes'),
    get: (_ns, name) => callApp('GetNodeDetail', name),
    remove: (_ns, name) => callApp('DeleteResource', 'node', '', name),
  },
})
