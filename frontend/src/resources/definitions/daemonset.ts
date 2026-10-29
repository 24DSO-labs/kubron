// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

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
import { t } from '../../i18n'

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
    { key: 'current', label: 'defs.columns.current' },
    { key: 'desired', label: 'defs.columns.desired' },
    { key: 'ready', label: 'defs.columns.ready' },
    { key: 'updated', label: 'defs.columns.upToDate' },
    { key: 'available', label: 'defs.columns.available' },
    { key: 'nodeSelector', label: 'defs.columns.nodeSelector', formatter: d => d.nodeSelector || '—' },
    ageCol,
  ],

  headSections: [workloadMetrics.daemonset],
  properties: [
    { label: 'defs.columns.selector', value: d => d.selector },
    { label: 'defs.columns.nodeSelector', value: d => d.nodeSelector },
    { label: 'defs.props.strategyType', value: d => d.strategy },
    { label: 'common.status', value: daemonsetStatus, badge: d => badgeClass(daemonsetStatus(d)) },
    controlledByProp,
    managedByProp,
  ],
  sections: [ImagesSection, TolerationsSection, NodeAffinitySection],
  tabs: [{ id: 'pods', label: 'defs.tabs.pods', component: PodsTab }],

  actions: [
    {
      id: 'restart',
      permission: Perm.Operate,
      label: 'defs.menu.restart',
      icon: 'restart',
      run: async (d, ctx) => {
        if (!(await ctx.confirm(t('defs.daemonSet.restartConfirm', { name: d.name })))) return
        return ctx.perform(
          () => callApp('RestartDaemonSet', d.namespace, d.name),
          t('defs.daemonSet.restarted', { name: d.name }),
        )
      },
    },
    { id: 'logs', label: 'defs.menu.viewLogs', icon: 'logs', permission: Perm.Read, run: (d, ctx) => ctx.openResourceLogs('daemonset', d.namespace, d.name) },
  ],

  api: {
    list: ns => callApp('GetDaemonSets', ns),
    remove: (ns, n) => callApp('DeleteResource', 'daemonset', ns, n),
  },
})