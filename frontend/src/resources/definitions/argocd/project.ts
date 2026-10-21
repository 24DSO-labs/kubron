import { defineResource } from '../../types'
import { callApp } from '../../wails'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoProjectInfo } from '../../../types/argocd'
import ArgoProjectAccessTab from '../../../components/resource/sections/ArgoProjectAccessTab.vue'
import ArgoProjectRolesTab from '../../../components/resource/sections/ArgoProjectRolesTab.vue'

/** Argo CD AppProject (читается через Kubernetes API; JWT-токены ролей не отдаются бэкендом) */
export const argoProject = defineResource<ArgoProjectInfo>({
  id: 'appproject',
  kind: 'AppProject',
  view: 'argo-projects',
  title: 'Projects',
  namespaced: true,
  nav: { group: 'argocd', label: 'Projects' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'description', label: 'Description', formatter: p => p.description || '—' },
    { key: 'sourceRepos', label: 'Repos', formatter: p => String(p.sourceRepos.length) },
    { key: 'destinations', label: 'Destinations', formatter: p => String(p.destinations.length) },
    { key: 'roles', label: 'Roles', formatter: p => String(p.roles.length) },
    ageCol,
  ],

  properties: [
    { label: 'Description', value: p => p.description, hideEmpty: true },
    { label: 'Source Repos', value: p => p.sourceRepos.length },
    { label: 'Destinations', value: p => p.destinations.length },
    { label: 'Roles', value: p => p.roles.length },
    { label: 'Orphaned Monitoring', value: p => (p.orphanedResourcesMonitoring ? 'Enabled' : null), hideEmpty: true },
  ],

  tabs: [
    { id: 'argoProjectAccess', label: 'Access', component: ArgoProjectAccessTab },
    { id: 'argoProjectRoles', label: 'Roles', component: ArgoProjectRolesTab },
  ],

  api: {
    list: ns => callApp('GetArgoProjects', ns) as Promise<ArgoProjectInfo[]>,
  },
})
