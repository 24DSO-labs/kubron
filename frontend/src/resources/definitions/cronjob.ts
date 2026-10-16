import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const cronJob = defineResource({
  id: 'cronjob',
  kind: 'CronJob',
  view: 'cronjobs',
  title: 'CronJobs',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
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
