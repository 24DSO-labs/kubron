import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import CronJobHistoryTab from '../../components/resource/sections/CronJobHistoryTab.vue'

export const cronJob = defineResource<k8s.CronJobInfo>({
  id: 'cronjob',
  kind: 'CronJob',
  view: 'cronjobs',
  title: 'CronJobs',
  namespaced: true,
  nav: { group: 'workloads', label: 'Cron Jobs' },

  autoRefresh: true,
  // icon: 'restart',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (c: any) => c.namespace ? { id: 'namespace', ns: '', name: c.namespace } : null },
    { key: 'schedule', label: 'Schedule' },
    {
      key: 'suspend',
      label: 'Suspend',
      formatter: c => (c.suspend ? 'true' : 'false'),
    },
    { key: 'active', label: 'Active' },
    { key: 'lastSchedule', label: 'Last schedule' },
    { key: 'nextExecution', label: 'Next execution' },
    { key: 'timeZone', label: 'Time zone' },
    ageCol,
  ],

  properties: [
    { label: 'Schedule', value: c => c.schedule },
    { label: 'Suspend', value: c => (c.suspend ? 'True' : 'False') },
    { label: 'Active', value: c => c.active },
    { label: 'Last Schedule', value: c => c.lastSchedule },
    { label: 'Next Execution', value: c => c.nextExecution },
    { label: 'Time Zone', value: c => c.timeZone },
  ],

  tabs: [
    { id: 'jobHistory', label: 'Job History', component: CronJobHistoryTab },
  ],

  actions: [
    {
      id: 'trigger',
      label: 'Trigger',
      icon: 'play',
      run: (c, ctx) =>
        ctx.perform(
          () => callApp('TriggerCronJob', c.namespace, c.name),
          `CronJob ${c.name}: triggered`,
        ),
    },
    {
      id: 'suspend',
      label: 'Suspend',
      icon: 'pause',
      when: c => !c.suspend,
      run: (c, ctx) =>
        ctx.perform(
          () => callApp('ToggleCronJobSuspend', c.namespace, c.name),
          `CronJob ${c.name}: suspended`,
        ),
    },
    {
      id: 'resume',
      label: 'Resume',
      icon: 'play',
      when: c => c.suspend,
      run: (c, ctx) =>
        ctx.perform(
          () => callApp('ToggleCronJobSuspend', c.namespace, c.name),
          `CronJob ${c.name}: resumed`,
        ),
    },
  ],

  api: {
    list: ns => callApp('GetCronJobs', ns),
    get: (ns, name) => callApp('GetCronJobDetail', ns, name),
    remove: (ns, name) => callApp('DeleteResource', 'cronjob', ns, name),
  },
})
