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
import PodAntiAffinitySection from '../../components/resource/sections/PodAntiAffinitySection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

function stsStatus(s: any): string {
  const desired = s.replicas ?? 0
  const ready = s.readyReplicas ?? 0
  if (desired === 0) return 'Scaled to 0'
  if (ready === desired) return `${desired}/${desired} Running`
  if (ready > 0) return `${ready}/${desired} Running`
  return `${ready}/${desired} Progressing`
}

export const statefulSet = defineResource<k8s.StatefulSetInfo>({
  id: 'statefulset',
  kind: 'StatefulSet',
  view: 'statefulsets',
  title: 'StatefulSets',
  namespaced: true,
  nav: { group: 'workloads', label: 'Stateful Sets' },

  autoRefresh: true,
  columns: [
    flatNameCol,
    { ...flatNsCol, link: (s: any) => s.namespace ? { id: 'namespace', ns: '', name: s.namespace } : null },
    {
      key: 'pods',
      label: 'Pods',
      formatter: s => `${s.readyReplicas ?? 0}/${s.replicas ?? 0}`,
      colorFn: s => stsStatus(s) === 'Scaled to 0' ? '#888' : (s.readyReplicas === s.replicas ? '#4ade80' : '#facc15'),
    },
    ageCol,
  ],

  headSections: [workloadMetrics.statefulset],
  properties: [
    { label: 'Selector', value: s => s.selector },
    { label: 'Node Selector', value: s => s.nodeSelector },
    { label: 'Status', value: stsStatus, badge: s => badgeClass(stsStatus(s)) },
    controlledByProp,
    managedByProp,
  ],
  sections: [ImagesSection, TolerationsSection, NodeAffinitySection, PodAntiAffinitySection],
  tabs: [
    { id: 'pods', label: 'Pods', component: PodsTab },
  ],

  actions: [
    { id: 'scale', permission: Perm.Operate, label: 'Scale', icon: 'scale', run: (d, ctx) => ctx.requestScale(d, 'StatefulSet') },
    {
      id: 'restart',
      permission: Perm.Operate,
      label: 'Restart',
      icon: 'restart',
      run: async (s, ctx) => {
        if (!(await ctx.confirm(`Restart statefulset ${s.name}?`))) return
        return ctx.perform(
          () => callApp('RestartStatefulSet', s.namespace, s.name),
          `StatefulSet ${s.name}: restart initiated`,
        )
      },
    },
    {
      id: 'logs',
      permission: Perm.Read,
      label: 'View logs',
      icon: 'logs',
      run: (s, ctx) => ctx.openResourceLogs('statefulset', s.namespace, s.name),
    },
  ],

  api: {
    list: ns => callApp('GetStatefulSets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'statefulset', ns, name),
  },
})