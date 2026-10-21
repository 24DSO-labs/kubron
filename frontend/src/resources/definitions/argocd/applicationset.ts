import { defineResource } from '../../types'
import { callApp } from '../../wails'
import { statusColor } from '../../status'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoApplicationSetInfo } from '../../../types/argocd'
import { shortRepo } from '../../argocd'
import ArgoConditionsSection from '../../../components/resource/sections/ArgoConditionsSection.vue'
import ArgoSetAppsTab from '../../../components/resource/sections/ArgoSetAppsTab.vue'

/** Argo CD ApplicationSet (читается через Kubernetes API; Delete не объявлен: он каскадно удаляет приложения) */
export const argoApplicationSet = defineResource<ArgoApplicationSetInfo>({
  id: 'applicationset',
  kind: 'ApplicationSet',
  view: 'argo-applicationsets',
  title: 'Application Sets',
  namespaced: true,
  nav: { group: 'argocd', label: 'Application Sets' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'generators', label: 'Generators', formatter: s => s.generators.join(', ') || '—' },
    { key: 'templateProject', label: 'Project', formatter: s => s.templateProject || '—' },
    { key: 'appCount', label: 'Apps' },
    {
      key: 'healthy',
      label: 'Status',
      formatter: s => (s.healthy ? 'Healthy' : 'Error'),
      colorFn: s => statusColor(s.healthy ? 'Healthy' : 'Error'),
    },
    ageCol,
  ],

  properties: [
    { label: 'Generators', value: s => s.generators.join(', ') || null, hideEmpty: true },
    { label: 'Template Project', value: s => s.templateProject, hideEmpty: true },
    { label: 'Template Repository', value: s => shortRepo(s.repoURL), hideEmpty: true },
    { label: 'Applications Sync', value: s => s.syncPolicy, hideEmpty: true },
    { label: 'Preserve on Deletion', value: s => (s.preserveResourcesOnDeletion ? 'Yes' : null), hideEmpty: true },
    { label: 'Applications', value: s => s.appCount },
  ],

  sections: [ArgoConditionsSection],
  tabs: [{ id: 'argoSetApps', label: 'Applications', component: ArgoSetAppsTab }],

  api: {
    list: ns => callApp('GetArgoApplicationSets', ns) as Promise<ArgoApplicationSetInfo[]>,
  },
})
