import { defineResource } from '../types'
import { Perm } from '../../types/auth'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import { badgeClass } from '../status'
import { workloadMetrics } from '../workloadMetrics'

import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import TolerationsSection from '../../components/resource/sections/TolerationsSection.vue'
import NodeAffinitySection from '../../components/resource/sections/NodeAffinitySection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

/** Статус DaemonSet на основе подов */
export function daemonsetStatus(ds: any): string {
  const desired = ds.desired ?? ds.desiredNumberScheduled ?? 0
  const ready = ds.ready ?? ds.numberReady ?? 0

  if (desired === 0) return 'Pending'
  if (ready === desired) return `${desired}/${desired} Running`
  if (ready > 0) return `${ready}/${desired} Running`
  return `${ready}/${desired} Progressing`
}

export const daemonSet = defineResource<k8s.DaemonSetInfo>({
  id: 'daemonset',
  kind: 'DaemonSet',
  view: 'daemonsets',
  title: 'DaemonSets',
  namespaced: true,
  nav: { group: 'workloads', label: 'Daemon Sets' },

  autoRefresh: true,
  columns: [
    flatNameCol,
    { ...flatNsCol, link: (d: any) => d.namespace ? { id: 'namespace', ns: '', name: d.namespace } : null },
    { key: 'current', label: 'Current' },
    { key: 'desired', label: 'Desired' },
    { key: 'ready', label: 'Ready' },
    { key: 'updated', label: 'Up-to-date' },
    { key: 'available', label: 'Available' },
    { key: 'nodeSelector', label: 'Node Selector', formatter: d => d.nodeSelector || '—' },
    ageCol,
  ],

  headSections: [workloadMetrics.daemonset],
  properties: [
    { label: 'Selector', value: d => d.selector },
    { label: 'Node Selector', value: d => d.nodeSelector },
    { label: 'Strategy Type', value: d => d.strategy },
    { label: 'Status', value: daemonsetStatus, badge: d => badgeClass(daemonsetStatus(d)) },
    controlledByProp,
    managedByProp,
  ],
  sections: [ImagesSection, TolerationsSection, NodeAffinitySection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  actions: [
    {
      id: 'restart',
      permission: Perm.Operate,
      label: 'Restart',
      icon: 'restart',
      run: async (d, ctx) => {
        if (!(await ctx.confirm(`Restart daemonset ${d.name}?`))) return
        return ctx.perform(
          () => callApp('RestartDaemonSet', d.namespace, d.name),
          `DaemonSet ${d.name}: restart initiated`,
        )
      },
    },
    { id: 'logs', label: 'View logs', icon: 'logs', permission: Perm.Read, run: (d, ctx) => ctx.openResourceLogs('daemonset', d.namespace, d.name) },
  ],

  api: {
    list: ns => callApp('GetDaemonSets', ns),
    remove: (ns, n) => callApp('DeleteResource', 'daemonset', ns, n),
  },
})