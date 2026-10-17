import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

import JobSpecSection from '../../components/resource/sections/JobSpecSection.vue'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'

function jobStatus(j: any): string {
  const conds: any[] = j?.status?.conditions ?? []
  const complete = conds.find((c: any) => c.type === 'Complete')
  const failed = conds.find((c: any) => c.type === 'Failed')
  if (failed?.status === 'True') return 'Failed'
  if (complete?.status === 'True') return 'Complete'
  return 'Running'
}

export const job = defineResource<k8s.JobInfo>({
  id: 'job',
  kind: 'Job',
  view: 'jobs',
  title: 'Jobs',
  namespaced: true,
  nav: { group: 'workloads', label: 'Jobs' },

  columns: [
    flatNameCol,
    flatNsCol,
    ageCol,
    {
      key: 'status',
      label: 'Status',
      formatter: jobStatus,
      colorFn: j => statusColor(jobStatus(j)),
    },
    { key: 'completions', label: 'Completions' },
    { key: 'duration', label: 'Duration' },
  ],

  properties: [
    { label: 'Status', value: j => jobStatus(j), badge: j => badgeClass(jobStatus(j)) },
  ],
  sections: [JobSpecSection, ImagesSection, ConditionsSection],

  actions: [
    {
      id: 'retry',
      label: 'Retry',
      icon: 'restart',
      run: (j, ctx) =>
        ctx.perform(
          () => callApp('RetryJob', j.namespace, j.name),
          `Job ${j.name} retry initiated`,
        ),
    },
  ],

  api: {
    list: ns => callApp('GetJobs', ns),
    remove: (ns, name) => callApp('DeleteJob', ns, name),
  },
})
