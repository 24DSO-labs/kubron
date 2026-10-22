import { defineResource } from '../../types'
import { callApp } from '../../wails'
import { badgeClass, statusColor } from '../../status'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoApplicationInfo } from '../../../types/argocd'
import { argoDestination, argoPathOrChart, shortRepo, shortRevision } from '../../argocd'
import ArgoConditionsSection from '../../../components/resource/sections/ArgoConditionsSection.vue'
import ArgoResourcesTab from '../../../components/resource/sections/ArgoResourcesTab.vue'
import ArgoHistoryTab from '../../../components/resource/sections/ArgoHistoryTab.vue'
import ArgoGraphTab from '../../../components/resource/sections/ArgoGraphTab.vue'
import ArgoSourcesTab from '../../../components/resource/sections/ArgoSourcesTab.vue'
import { Perm } from '../../../types/auth'
import { argoLiveTick } from '../../../composables/useArgoLive'
import { openArgoUI, refreshArgoApps, requestArgoSync, terminateArgoOperation, } from '../../argoActions'

const syncPolicy = (a: ArgoApplicationInfo) =>
  a.autoSync
    ? ['Automated', a.prune ? 'prune' : '', a.selfHeal ? 'self-heal' : ''].filter(Boolean).join(', ')
    : 'Manual'

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
    { key: 'project', label: 'Project' },
    {
      key: 'syncStatus',
      label: 'Sync',
      formatter: a => a.syncStatus || 'Unknown',
      colorFn: a => statusColor(a.syncStatus),
    },
    {
      key: 'healthStatus',
      label: 'Health',
      formatter: a => a.healthStatus || 'Unknown',
      colorFn: a => statusColor(a.healthStatus),
    },
    { key: 'repoURL', label: 'Repo', formatter: a => shortRepo(a.repoURL) || '—' },
    { key: 'path', label: 'Path / Chart', formatter: a => argoPathOrChart(a) || '—' },
    { key: 'revision', label: 'Revision', formatter: a => shortRevision(a.revision) || '—' },
    { key: 'destNamespace', label: 'Destination', formatter: a => argoDestination(a) || '—' },
    { key: 'lastSyncAt', label: 'Last sync', formatter: a => a.lastSyncAt || '—' },
    ageCol,
  ],

  properties: [
    { label: 'Project', value: a => a.project },
    { label: 'Sync Status', value: a => a.syncStatus, badge: a => badgeClass(a.syncStatus) },
    { label: 'Health Status', value: a => a.healthStatus, badge: a => badgeClass(a.healthStatus) },
    { label: 'Message', value: a => a.message, hideEmpty: true },
    { label: 'Revision', value: a => a.revision, hideEmpty: true },
    { label: 'Repository', value: a => shortRepo(a.repoURL), hideEmpty: true },
    { label: 'Path / Chart', value: a => argoPathOrChart(a), hideEmpty: true },
    { label: 'Target Revision', value: a => a.targetRevision, hideEmpty: true },
    { label: 'Destination', value: a => argoDestination(a) },
    { label: 'Sync Policy', value: syncPolicy },
    {
      label: 'Last Sync',
      value: a => [a.lastSyncPhase, a.lastSyncAt].filter(Boolean).join(' · ') || null,
      hideEmpty: true,
    },
    { label: 'Operation', value: a => (a.operationOn ? 'In progress' : null), hideEmpty: true },
  ],

  sections: [ArgoConditionsSection],

  tabs: [
    { id: 'argoResources', label: 'Resources', component: ArgoResourcesTab },
    { id: 'argoHistory', label: 'History', component: ArgoHistoryTab },
    { id: 'argoGraph', label: 'Graph', component: ArgoGraphTab },
    { id: 'argoSources', label: 'Sources', component: ArgoSourcesTab },
  ],
  actions: [
    {
      id: 'sync', label: 'Sync', icon: 'refresh', permission: Perm.Operate,
      when: a => !a.operationOn,
      run: a => requestArgoSync([a as ArgoApplicationInfo]),
    },
    {
      id: 'refresh', label: 'Refresh', icon: 'restart', permission: Perm.Operate,
      run: a => refreshArgoApps([a as ArgoApplicationInfo], false),
    },
    {
      id: 'hardrefresh', label: 'Hard Refresh', icon: 'restart', permission: Perm.Operate,
      run: a => refreshArgoApps([a as ArgoApplicationInfo], true),
    },
    {
      id: 'terminate', label: 'Terminate operation', icon: 'stop', danger: true, permission: Perm.Operate,
      when: a => !!a.operationOn,
      run: a => terminateArgoOperation([a as ArgoApplicationInfo]),
    },
    {
      id: 'argoui', label: 'Open Argo UI', icon: 'open-browser', permission: Perm.PortForward,
      run: () => openArgoUI(),
    },
  ],
  api: {
    // Пустой namespace = все namespace (Application может жить в любом)
    list: ns => callApp('GetArgoApplications', ns) as Promise<ArgoApplicationInfo[]>,
    // Мутирующих операций в MVP нет: Delete не объявлен (remove отсутствует)
  },
})
