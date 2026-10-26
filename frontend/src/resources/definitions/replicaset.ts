// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import { Perm } from '../../types/auth'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, managedByProp } from './common'
import { badgeClass } from '../status'
import { podStatusSummary } from '../utils'
import { workloadMetrics } from '../workloadMetrics'

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

export const replicaSet = defineResource<k8s.ReplicaSetInfo>({
  id: 'replicaset',
  kind: 'ReplicaSet',
  view: 'replicasets',
  title: 'ReplicaSets',
  namespaced: true,
  nav: { group: 'workloads', label: 'Replica Sets' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (r: any) => r.namespace ? { id: 'namespace', ns: '', name: r.namespace } : null },
    { key: 'desired', label: 'Desired' },
    { key: 'current', label: 'Current' },
    { key: 'ready', label: 'Ready' },
    ageCol,
  ],

  headSections: [workloadMetrics.replicaset],
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
    managedByProp,
  ],
  sections: [ImagesSection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  actions: [
    { id: 'scale', permission: Perm.Operate, label: 'Scale', icon: 'scale', run: (r, ctx) => ctx.requestScale(r, 'ReplicaSet') },
  ],

  api: {
    list: ns => callApp('GetReplicaSets', ns),
    remove: (ns, n) => callApp('DeleteResource', 'replicaset', ns, n),
  },
})