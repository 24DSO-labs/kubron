import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'
import scaleIcon from '../../assets/icons/scale.svg'
import restartIcon from '../../assets/icons/restart.svg'
import logsIcon from '../../assets/icons/logs.svg'
import DeploymentMetricsSection from '../../components/resource/sections/DeploymentMetricsSection.vue'
import ImagesSection from '../../components/resource/sections/ImagesSection.vue'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import PodAntiAffinitySection from '../../components/resource/sections/PodAntiAffinitySection.vue'
import PodsTab from '../../components/resource/sections/PodsTab.vue'
import RevisionsTab from '../../components/resource/sections/RevisionsTab.vue'

/** Статус по conditions (Available + Progressing) */
export function deploymentStatus(dep: any): string {
  const conds: any[] = dep?.conditions ?? []
  if (conds.length === 0) return 'Pending'
  const available = conds.find(c => c.type === 'Available')
  const progressing = conds.find(c => c.type === 'Progressing')

  if (available?.status === 'True' && progressing?.status === 'True') return 'Running'
  if (progressing?.status === 'False') return progressing.reason || 'Failed'
  if (progressing?.reason) return progressing.reason
  return 'Pending'
}

function replicasSummary(d: any): string {
  const desired = d.replicas ?? 0
  const ready = d.readyReplicas ?? 0
  return `${desired} desired, ${d.updatedReplicas ?? 0} updated, ${desired} total, ${ready} available, ${Math.max(0, desired - ready)} unavailable`
}

export const deployment = defineResource<k8s.DeploymentInfo>({
  id: 'deployment',
  kind: 'Deployment',
  view: 'deployments',
  title: 'Deployments',
  namespaced: true,
  nav: { group: 'workloads', label: 'Deployments' },

  columns: [
    flatNameCol,
    flatNsCol,
    {
      key: 'pods',
      label: 'Pods',
      formatter: d => `${d.readyReplicas}/${d.replicas}`,
      colorFn: d => (d.readyReplicas === d.replicas ? '#4ade80' : '#facc15'),
    },
    { key: 'updatedReplicas', label: 'Replicas' },
    ageCol,
    {
      key: 'status',
      label: 'Status',
      formatter: deploymentStatus,
      colorFn: d => statusColor(deploymentStatus(d)),
    },
  ],

  headSections: [DeploymentMetricsSection],
  properties: [
    { label: 'Replicas', value: replicasSummary },
    { label: 'Selector', value: d => d.selector },
    { label: 'Node Selector', value: d => d.nodeSelector },
    { label: 'Strategy Type', value: d => d.strategy },
    { label: 'Status', value: deploymentStatus, badge: d => badgeClass(deploymentStatus(d)) },
  ],
  sections: [ImagesSection, ConditionsSection, PodAntiAffinitySection],
  tabs: [
    { id: 'revisions', label: 'Revisions', component: RevisionsTab },
    { id: 'pods', label: 'Pods', component: PodsTab },
  ],

  actions: [
    { id: 'scale', label: 'Scale', icon: scaleIcon, run: (d, ctx) => ctx.requestScale(d) },
    {
      id: 'restart',
      label: 'Restart',
      icon: restartIcon,
      run: async (d, ctx) => {
        if (!(await ctx.confirm(`Restart deployment ${d.name}?`))) return
        return ctx.perform(
          () => callApp('RestartDeployment', d.namespace, d.name),
          `Deployment ${d.name}: restart initiated`,
        )
      },
    },
    // Название "View logs", а не "Logs": список особо обрабатывает пункт с меткой "Logs" (подменю контейнеров pod)
    { id: 'logs', label: 'View logs', icon: logsIcon, run: (d, ctx) => ctx.openResourceLogs('deployment', d.namespace, d.name) },
  ],

  api: {
    list: ns => callApp('GetDeployments', ns),
    remove: (ns, n) => callApp('DeleteDeployment', ns, n),
  },
})
