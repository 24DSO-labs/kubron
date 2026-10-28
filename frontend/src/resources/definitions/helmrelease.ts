// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { helmLiveTick, startHelmLive } from '../../composables/useHelmLive'
import { runHelmOp } from '../../composables/useHelmOps'
import { flatNameCol, flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'
import { helmContentTab } from '../helmTabs'
import HelmResourcesTab from '../../components/resource/sections/HelmResourcesTab.vue'
import HelmHistoryTab from '../../components/resource/sections/HelmHistoryTab.vue'
import { Perm } from '../../types/auth'
import HelmUpgradeTab from '../../components/resource/sections/HelmUpgradeTab.vue'

export const helmRelease = defineResource<k8s.HelmReleaseInfo, k8s.HelmReleaseDetail>({
  id: 'helmrelease',
  kind: 'Helm Release',
  view: 'helm-releases',
  title: 'Helm Releases',
  namespaced: false,
  nav: { group: 'helm', label: 'Releases' },

  autoRefresh: true,
  // helm-релиз — агрегат из secret-ов, не отдельный k8s-ресурс: без YAML и удаления
  // (uninstall появится на этапе H2 как отдельное действие)
  editable: false,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (h: any) => h.namespace ? { id: 'namespace', ns: '', name: h.namespace } : null },
    { key: 'chartVersion', label: 'Chart Version' },
    { key: 'appVersion', label: 'App Version' },
    { key: 'revision', label: 'Revision' },
    { key: 'status', label: 'Status', colorFn: r => statusColor(r.status) },
    {
      key: 'latestVersion',
      label: 'Update',
      formatter: r => (r.latestVersion ? `${r.latestVersion} (${r.latestRepo})` : ''),
      colorFn: () => '#facc15',
      sortValue: r => r.latestVersion || null,
    },
    { key: 'updated', label: 'Updated' },
  ],

  properties: [
    { label: 'Status', value: r => r.status, badge: r => badgeClass(r.status) },
    { label: 'Chart', value: r => r.chart },
    { label: 'Chart Version', value: r => r.chartVersion },
    { label: 'Storage', value: r => (r.storage === 'configmap' ? 'ConfigMap' : 'Secret') },
    { label: 'Update available', value: r => (r.latestVersion ? `${r.latestVersion} (${r.latestRepo})` : '—') },
    { label: 'App Version', value: r => r.appVersion },
    { label: 'Revision', value: r => r.revision },
    { label: 'Updated', value: r => r.updated },
    { label: 'Description', value: r => r.description },
  ],

  tabs: [
    { id: 'helmValues', label: 'Values', component: helmContentTab('values') },
    { id: 'helmNotes', label: 'Notes', component: helmContentTab('notes') },
    { id: 'helmManifest', label: 'Manifest', component: helmContentTab('manifest') },
    { id: 'helmEdit', label: 'Edit values', component: HelmUpgradeTab },
    { id: 'helmResources', label: 'Resources', component: HelmResourcesTab },
    { id: 'helmHistory', label: 'History', component: HelmHistoryTab },

  ],
actions: [
    {
      id: 'uninstall',
      label: 'Uninstall',
      icon: 'delete',
      danger: true,
      permission: Perm.Write,
      run: async (r, ctx) => {
        if (!(await ctx.confirm(
          `Uninstall release ${r.namespace}/${r.name}? All resources created by the release will be deleted from the cluster.`,
        ))) return
        try {
          await runHelmOp({ op: 'uninstall', namespace: r.namespace, release: r.name }, id =>
            callApp('UninstallHelmRelease', r.namespace, r.name, false, id))
          ctx.notifySuccess(`Release ${r.name} uninstalled`)
          ctx.closeDetail()
          await ctx.reload()
        } catch (e: any) {
          ctx.notifyError(`Uninstall: ${e?.message || e}`)
        }
      },
    },
  ],
  // helm:changed (informer): список и открытая панель обновляются сразу, без ожидания таймера
  refreshOn: () => { startHelmLive(); return helmLiveTick.value },
  liveDetail: true,
  api: {
    list: () => callApp('GetHelmReleases'),
    // history и resources приходят отсюда и дополняют элемент списка
    get: (ns, name) => callApp('GetHelmReleaseDetail', ns, name),
  },
})