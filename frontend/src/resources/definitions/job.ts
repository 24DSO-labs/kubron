// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import { Perm } from '../../types/auth'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import { statusColor } from '../status'
import { workloadMetrics } from '../workloadMetrics'

import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import JobStatsSection from '../../components/resource/sections/JobStatsSection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

const jobStatus = (j: any): string => j.status || 'Running'

/** conditions теперь объекты; в колонке показываем только активные (status=True) */
const jobConditions = (j: any): string => {
  const active = (j.conditions ?? []).filter((c: any) => c.status === 'True').map((c: any) => c.type)
  return active.length ? active.join(', ') : jobStatus(j)
}

export const job = defineResource<k8s.JobInfo>({
  id: 'job',
  kind: 'Job',
  view: 'jobs',
  title: 'Jobs',
  namespaced: true,
  nav: { group: 'workloads', label: 'Jobs' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (j: any) => j.namespace ? { id: 'namespace', ns: '', name: j.namespace } : null },
    { key: 'completions', label: 'Completions' },
    ageCol,
    {
      key: 'conditions',
      label: 'Conditions',
      formatter: jobConditions,
      colorFn: j => statusColor(jobStatus(j)),
    },
  ],

  // Порядок панели: метрики -> Labels -> Annotations -> properties -> sections
  headSections: [workloadMetrics.job],
  properties: [
    { label: 'Selector', value: j => j.selector },
    controlledByProp,
    managedByProp,
  ],
  sections: [ImagesSection, ConditionsSection, JobStatsSection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  actions: [
    {
      id: 'retry',
      permission: Perm.Operate,
      label: 'Retry',
      icon: 'restart',
      run: async (j, ctx) => {
        if (!(await ctx.confirm(`Re-run job ${j.name}? The Job and its pods (with logs) will be deleted and created again.`))) return
        return ctx.perform(
          () => callApp('RetryJob', j.namespace, j.name),
          `Job ${j.name} retry initiated`,
        )
      },
    },
  ],

  api: {
    list: ns => callApp('GetJobs', ns),
    remove: (ns, name) => callApp('DeleteResource', 'job', ns, name),
  },
})