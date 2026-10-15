import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import restartIcon from '../../assets/icons/restart.svg'
import logsIcon from '../../assets/icons/logs.svg'
import DaemonSetMetricsSection from '../../components/resource/sections/DaemonSetMetricsSection.vue'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import TolerationsSection from '../../components/resource/sections/TolerationsSection.vue'
import NodeAffinitySection from '../../components/resource/sections/NodeAffinitySection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'

export const daemonSet = defineResource({
  id: 'daemonset',
  kind: 'DaemonSet',
  view: 'daemonsets',
  title: 'DaemonSets',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'desired', label: 'Desired' },
    { key: 'current', label: 'Current' },
    { key: 'ready', label: 'Ready' },
    { key: 'available', label: 'Available' },
    ageCol,
  ],

  headSections: [DaemonSetMetricsSection],
  properties: [
    { label: 'Selector', value: d => d.selector },
    { label: 'Node Selector', value: d => d.nodeSelector },
    { label: 'Strategy Type', value: d => d.strategy },
    {
      label: 'Pod Status',
      value: d => `${d.ready ?? d.numberReady ?? 0}/${d.desired ?? d.desiredNumberScheduled ?? 0} ready`,
    },
  ],
  sections: [ImagesSection, TolerationsSection, NodeAffinitySection],
  tabs: [{ id: 'pods', label: 'Pods', component: PodsTab }],

  actions: [
    {
      id: 'restart',
      label: 'Restart',
      icon: restartIcon,
      run: (d, ctx) => {
        if (!ctx.confirm(`Restart daemonset ${d.name}?`)) return
        // Нужен Go-метод RestartDaemonSet (annotation kubectl.kubernetes.io/restartedAt)
        return ctx.perform(
          () => callApp('RestartDaemonSet', d.namespace, d.name),
          `DaemonSet ${d.name}: restart initiated`,
        )
      },
    },
    { id: 'logs', label: 'View logs', icon: logsIcon, run: (d, ctx) => ctx.openResourceLogs('daemonset', d.namespace, d.name) },
  ],

  api: {
    list: ns => callApp('GetDaemonSets', ns),
    remove: (ns, n) => callApp('DeleteDaemonSet', ns, n),
  },
})
