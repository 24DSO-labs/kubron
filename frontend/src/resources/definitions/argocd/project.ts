// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../../types'
import { t } from '../../../i18n'
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
    { key: 'description', label: 'defs.props.description', formatter: p => p.description || '—' },
    { key: 'sourceRepos', label: 'argo.columns.repos', formatter: p => String(p.sourceRepos.length) },
    { key: 'destinations', label: 'argo.project.destinations', formatter: p => String(p.destinations.length) },
    { key: 'roles', label: 'defs.columns.roles', formatter: p => String(p.roles.length) },
    ageCol,
  ],

  properties: [
    { label: 'defs.props.description', value: p => p.description, hideEmpty: true },
    { label: 'argo.props.sourceRepos', value: p => p.sourceRepos.length },
    { label: 'argo.project.destinations', value: p => p.destinations.length },
    { label: 'defs.columns.roles', value: p => p.roles.length },
    { label: 'argo.props.orphanedMonitoring', value: p => (p.orphanedResourcesMonitoring ? t('common.enabled') : null), hideEmpty: true },
  ],

  tabs: [
    { id: 'argoProjectAccess', label: 'argo.tabs.access', component: ArgoProjectAccessTab },
    { id: 'argoProjectRoles', label: 'defs.columns.roles', component: ArgoProjectRolesTab },
  ],

  api: {
    list: ns => callApp('GetArgoProjects', ns) as Promise<ArgoProjectInfo[]>,
  },
})
