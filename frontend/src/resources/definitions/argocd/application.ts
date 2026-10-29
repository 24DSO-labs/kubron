// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../../types'
import { t } from '../../../i18n'
import { callApp } from '../../wails'
import { badgeClass, statusColor } from '../../status'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoApplicationInfo } from '../../../types/argocd'
import { argoDestination, argoPathOrChart, shortRepo, shortRevision } from '../../argocd'
import ArgoConditionsSection from '../../../components/resource/sections/ArgoConditionsSection.vue'
import ArgoResourcesTab from '../../../components/resource/sections/ArgoResourcesTab.vue'
import ArgoHistoryTab from '../../../components/resource/sections/ArgoHistoryTab.vue'
import ArgoGraphTab from '../../../components/resource/sections/ArgoGraphTab.vue'
import ArgoDiffTab from '../../../components/resource/sections/ArgoDiffTab.vue'
import ArgoSourcesTab from '../../../components/resource/sections/ArgoSourcesTab.vue'
import { Perm } from '../../../types/auth'
import { argoLiveTick } from '../../../composables/useArgoLive'
import { openArgoUI, refreshArgoApps, requestArgoDelete, setArgoAutoSync, requestArgoSync, terminateArgoOperation, } from '../../argoActions'

const syncPolicy = (a: ArgoApplicationInfo) =>
  a.autoSync
    ? [t('argo.syncPolicy.automated'), a.prune ? t('argo.syncPolicy.prune') : '', a.selfHeal ? t('argo.syncPolicy.selfHeal') : '']
      .filter(Boolean).join(', ')
    : t('argo.syncPolicy.manual')

/** Argo CD Application (read-only, MVP: только Kubernetes API, CRD argoproj.io/v1alpha1) */
export const argoApplication = defineResource<ArgoApplicationInfo>({
  id: 'application',
  kind: 'Application',
  view: 'argo-applications',
  title: 'Applications',
  namespaced: true,
  nav: { group: 'argocd', label: 'Applications' },

  autoRefresh: true,
  // Список перезагружается по событию argocd:changed (informer), таймер остаётся страховкой
  refreshOn: () => argoLiveTick.value,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'project', label: 'argo.columns.project' },
    {
      key: 'syncStatus',
      label: 'argo.columns.sync',
      formatter: a => a.syncStatus || t('common.unknown'),
      colorFn: a => statusColor(a.syncStatus),
    },
    {
      key: 'healthStatus',
      label: 'argo.columns.health',
      formatter: a => a.healthStatus || t('common.unknown'),
      colorFn: a => statusColor(a.healthStatus),
    },
    { key: 'repoURL', label: 'argo.columns.repo', formatter: a => shortRepo(a.repoURL) || '—' },
    { key: 'path', label: 'argo.sources.pathOrChart', formatter: a => argoPathOrChart(a) || '—' },
    { key: 'revision', label: 'defs.columns.revision', formatter: a => shortRevision(a.revision) || '—' },
    { key: 'destNamespace', label: 'argo.columns.destination', formatter: a => argoDestination(a) || '—' },
    { key: 'lastSyncAt', label: 'argo.columns.lastSync', formatter: a => a.lastSyncAt || '—' },
    ageCol,
  ],

  properties: [
    { label: 'argo.columns.project', value: a => a.project },
    { label: 'argo.props.syncStatus', value: a => a.syncStatus, badge: a => badgeClass(a.syncStatus) },
    { label: 'argo.props.healthStatus', value: a => a.healthStatus, badge: a => badgeClass(a.healthStatus) },
    { label: 'defs.columns.message', value: a => a.message, hideEmpty: true },
    { label: 'defs.columns.revision', value: a => a.revision, hideEmpty: true },
    { label: 'argo.sources.repository', value: a => shortRepo(a.repoURL), hideEmpty: true },
    { label: 'argo.sources.pathOrChart', value: a => argoPathOrChart(a), hideEmpty: true },
    { label: 'argo.sources.targetRevision', value: a => a.targetRevision, hideEmpty: true },
    { label: 'argo.columns.destination', value: a => argoDestination(a) },
    { label: 'argo.props.syncPolicy', value: syncPolicy },
    {
      label: 'argo.props.lastSync',
      value: a => [a.lastSyncPhase, a.lastSyncAt].filter(Boolean).join(' · ') || null,
      hideEmpty: true,
    },
    { label: 'argo.props.operation', value: a => (a.operationOn ? t('argo.props.inProgress') : null), hideEmpty: true },
  ],

  sections: [ArgoConditionsSection],

  tabs: [
    { id: 'argoResources', label: 'defs.tabs.resources', component: ArgoResourcesTab },
    { id: 'argoHistory', label: 'defs.tabs.history', component: ArgoHistoryTab },
    { id: 'argoGraph', label: 'argo.tabs.graph', component: ArgoGraphTab },
    { id: 'argoDiff', label: 'argo.tabs.diff', component: ArgoDiffTab },
    { id: 'argoSources', label: 'argo.tabs.sources', component: ArgoSourcesTab },
  ],
  actions: [
    {
      id: 'sync', label: 'argo.menu.sync', icon: 'refresh', permission: Perm.Operate,
      when: a => !a.operationOn,
      run: a => requestArgoSync([a as ArgoApplicationInfo]),
    },
    {
      id: 'refresh', label: 'common.refresh', icon: 'restart', permission: Perm.Operate,
      run: a => refreshArgoApps([a as ArgoApplicationInfo], false),
    },
    {
      id: 'hardrefresh', label: 'argo.menu.hardRefresh', icon: 'restart', permission: Perm.Operate,
      run: a => refreshArgoApps([a as ArgoApplicationInfo], true),
    },
    {
      id: 'terminate', label: 'argo.menu.terminate', icon: 'stop', danger: true, permission: Perm.Operate,
      when: a => !!a.operationOn,
      run: a => terminateArgoOperation([a as ArgoApplicationInfo]),
    },
    {
      id: 'argoui', label: 'argo.menu.openUi', icon: 'open-browser', permission: Perm.PortForward,
      run: () => openArgoUI(),
    },
        {
      id: 'disableAutoSync', label: 'argo.menu.disableAutoSync', icon: 'pause', permission: Perm.Write,
      when: a => !!a.autoSync,
      run: a => setArgoAutoSync([a as ArgoApplicationInfo], false),
    },
    {
      id: 'enableAutoSync', label: 'argo.menu.enableAutoSync', icon: 'play', permission: Perm.Write,
      when: a => !a.autoSync,
      run: a => setArgoAutoSync([a as ArgoApplicationInfo], true),
    },
    {
      id: 'delete', label: 'common.delete', icon: 'delete', danger: true, permission: Perm.Write,
      run: a => requestArgoDelete(a as ArgoApplicationInfo),
    },
  ],
  api: {
    // Пустой namespace = все namespace (Application может жить в любом)
    list: ns => callApp('GetArgoApplications', ns) as Promise<ArgoApplicationInfo[]>,
    // remove не объявлен намеренно: удаление идёт через действие delete (выбор cascade)
  },
})
