import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const cronJob = defineResource<k8s.CronJobInfo>({
  id: 'cronjob',
  kind: 'CronJob',
  view: 'cronjobs',
  title: 'CronJobs',
  namespaced: true,
  nav: { group: 'workloads', label: 'Cron Jobs' },
  
  autoRefresh: true,icon: 'restart',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (c: any) => c.namespace ? { id: 'namespace', ns: '', name: c.namespace } : null },
    { key: 'schedule', label: 'Schedule' },
    {
      key: 'suspend',
      label: 'Suspend',
      formatter: c => (c.suspend ? 'True' : 'False'),
    },
    { key: 'active', label: 'Active' },
    { key: 'lastSchedule', label: 'Last Schedule' },
    ageCol,
  ],

  properties: [
    { label: 'Schedule', value: c => c.schedule },
    { label: 'Suspend', value: c => (c.suspend ? 'True' : 'False') },
    { label: 'Active', value: c => c.active },
    { label: 'Last Schedule', value: c => c.lastSchedule },
  ],

  api: {
    list: ns => callApp('GetCronJobs', ns),
    remove: (ns, name) => callApp('DeleteCronJob', ns, name),
  },
})
