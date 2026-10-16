import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'
import restartIcon from '../../assets/icons/restart.svg'
import logsIcon from '../../assets/icons/logs.svg'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

function stsStatus(s: any): string {
  const desired = s.replicas ?? 0
  const ready = s.readyReplicas ?? 0
  if (desired === 0) return 'Scaled to 0'
  if (ready === desired) return 'Running'
  return 'Pending'
}

export const statefulSet = defineResource({
  id: 'statefulset',
  kind: 'StatefulSet',
  view: 'statefulsets',
  title: 'StatefulSets',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    {
      key: 'pods',
      label: 'Pods',
      formatter: s => `${s.readyReplicas ?? 0}/${s.replicas ?? 0}`,
      colorFn: s => (s.readyReplicas === s.replicas ? '#4ade80' : '#facc15'),
    },
    { key: 'replicas', label: 'Replicas' },
    ageCol,
    {
      key: 'status',
      label: 'Status',
      formatter: stsStatus,
      colorFn: s => statusColor(stsStatus(s)),
    },
  ],

  properties: [
    { label: 'Selector', value: s => s.selector },
    { label: 'Replicas', value: s => `${s.replicas ?? 0} / ${s.readyReplicas ?? 0}` },
    { label: 'Strategy Type', value: s => s.strategy || 'RollingUpdate' },
    { label: 'Status', value: stsStatus, badge: s => badgeClass(stsStatus(s)) },
  ],
  sections: [ImagesSection],
  tabs: [
    { id: 'pods', label: 'Pods', component: PodsTab },
  ],

  actions: [
    {
      id: 'restart',
      label: 'Restart',
      icon: restartIcon,
      run: (s, ctx) => {
        if (!ctx.confirm(`Restart statefulset ${s.name}?`)) return
        return ctx.perform(
          () => callApp('RestartStatefulSet', s.namespace, s.name),
          `StatefulSet ${s.name}: restart initiated`,
        )
      },
    },
    {
      id: 'logs',
      label: 'View logs',
      icon: logsIcon,
      run: (s, ctx) => ctx.openResourceLogs('statefulset', s.namespace, s.name),
    },
  ],

  api: {
    list: ns => callApp('GetStatefulSets', ns),
    remove: (ns, name) => callApp('DeleteStatefulSet', ns, name),
  },
})
