import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass } from '../status'

import ReplicaSetMetricsSection from '../../components/resource/sections/ReplicaSetMetricsSection.vue'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

/** Маппинг kind -> resource id для ссылок (владелец ReplicaSet — обычно Deployment) */
const kindToId: Record<string, string> = {
  Deployment: 'deployment',
  DaemonSet: 'daemonset',
  StatefulSet: 'statefulset',
  ReplicaSet: 'replicaset',
  Job: 'job',
  CronJob: 'cronjob',
}

function controlledByLink(r: any): { id: string; ns: string; name: string } | null {
  const cb = r.controlledBy
  if (!cb) return null
  const [kind, name] = cb.split('/')
  if (!kind || !name) return null
  const id = kindToId[kind]
  if (!id) return null
  return { id, ns: r.namespace, name }
}

/** Статус ReplicaSet на основе подов */
export function replicaSetStatus(rs: any): string {
  const desired = rs.desired ?? 0
  const ready = rs.ready ?? 0
  if (desired === 0) return 'Scaled to 0'
  if (ready === desired) return `${ready}/${desired} Running`
  if (ready > 0) return `${ready}/${desired} Running`
  return `${ready}/${desired} Progressing`
}

/** Pod Status из счётчиков фаз: "Running: 1" / "Running: 1, Pending: 2" */
const PHASE_ORDER = ['Running', 'Pending', 'Succeeded', 'Failed', 'Unknown']
function podStatusSummary(rs: any): string {
  const statuses = rs.podStatuses ?? {}
  const entries = Object.entries(statuses).filter(([, n]) => (n as number) > 0)
  if (!entries.length) return ''
  entries.sort((a, b) => {
    const ia = PHASE_ORDER.indexOf(a[0])
    const ib = PHASE_ORDER.indexOf(b[0])
    return (ia === -1 ? PHASE_ORDER.length : ia) - (ib === -1 ? PHASE_ORDER.length : ib)
  })
  return entries.map(([phase, n]) => `${phase}: ${n}`).join(', ')
}

export const replicaSet = defineResource<k8s.ReplicaSetInfo>({
  id: 'replicaset',
  kind: 'ReplicaSet',
  view: 'replicasets',
  title: 'ReplicaSets',
  namespaced: true,
  nav: { group: 'workloads', label: 'Replica Sets' },
  
  autoRefresh: true,
  // icon: 'service',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (r: any) => r.namespace ? { id: 'namespace', ns: '', name: r.namespace } : null },
    { key: 'desired', label: 'Desired' },
    { key: 'current', label: 'Current' },
    { key: 'ready', label: 'Ready' },
    ageCol,
  ],

  headSections: [ReplicaSetMetricsSection],
  properties: [
    { label: 'Replicas', value: r => `${r.current ?? 0} current / ${r.desired ?? 0} desired` },
    {
      label: 'Controlled By',
      value: r => {
        const cb = r.controlledBy
        if (!cb) return null
        const [kind] = cb.split('/')
        const hasLink = kind && kindToId[kind]
        return hasLink ? kind + ' ' : cb
      },
      linkLabel: r => {
        const cb = r.controlledBy
        if (!cb) return null
        const [, name] = cb.split('/')
        return name || null
      },
      link: controlledByLink,
    },
    { label: 'Selector', value: r => r.selector },
    { label: 'Pod Status', value: podStatusSummary },
    { label: 'Status', value: replicaSetStatus, badge: r => badgeClass(replicaSetStatus(r)) },
  ],
  sections: [ImagesSection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  actions: [
    { id: 'scale', label: 'Scale', icon: 'scale', run: (r, ctx) => ctx.requestScale(r, 'ReplicaSet') },
  ],

  api: {
    list: ns => callApp('GetReplicaSets', ns),
    remove: (ns, n) => callApp('DeleteReplicaSet', ns, n),
  },
})
