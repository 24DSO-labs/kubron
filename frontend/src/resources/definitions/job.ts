import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

import JobSpecSection from '../../components/resource/sections/JobSpecSection.vue'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import JobWarningCell from '../../components/resource/sections/JobWarningCell.vue'

const jobStatus = (j: any): string => j.status || 'Running'
const jobConditions = (j: any): string =>
  j.conditions?.length ? j.conditions.join(', ') : jobStatus(j)



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

  properties: [
    { label: 'Status', value: j => jobStatus(j), badge: j => badgeClass(jobStatus(j)) },
    { label: 'Completions', value: j => j.completions },
    { label: 'Duration', value: j => j.duration },
    { label: 'Conditions', value: j => jobConditions(j) },
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
