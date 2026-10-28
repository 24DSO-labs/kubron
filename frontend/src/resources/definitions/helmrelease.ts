// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'
import { helmContentTab } from '../helmTabs'
import HelmResourcesTab from '../../components/resource/sections/HelmResourcesTab.vue'
import HelmHistoryTab from '../../components/resource/sections/HelmHistoryTab.vue'

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
    { key: 'chartVersion', label: 'defs.columns.chartVersion' },
    { key: 'appVersion', label: 'defs.columns.appVersion' },
    { key: 'revision', label: 'defs.columns.revision' },
    { key: 'status', label: 'common.status', colorFn: r => statusColor(r.status) },
    { key: 'updated', label: 'defs.columns.updated' },
  ],

  properties: [
    { label: 'common.status', value: r => r.status, badge: r => badgeClass(r.status) },
    { label: 'defs.props.chart', value: r => r.chart },
    { label: 'defs.columns.chartVersion', value: r => r.chartVersion },
    { label: 'defs.columns.appVersion', value: r => r.appVersion },
    { label: 'defs.columns.revision', value: r => r.revision },
    { label: 'defs.columns.updated', value: r => r.updated },
    { label: 'defs.props.description', value: r => r.description },
  ],

  tabs: [
    { id: 'helmValues', label: 'defs.tabs.values', component: helmContentTab('values') },
    { id: 'helmNotes', label: 'defs.tabs.notes', component: helmContentTab('notes') },
    { id: 'helmManifest', label: 'defs.tabs.manifest', component: helmContentTab('manifest') },
    { id: 'helmResources', label: 'defs.tabs.resources', component: HelmResourcesTab },
    { id: 'helmHistory', label: 'defs.tabs.history', component: HelmHistoryTab },
  ],

  api: {
    list: () => callApp('GetHelmReleases'),
    // history и resources приходят отсюда и дополняют элемент списка
    get: (ns, name) => callApp('GetHelmReleaseDetail', ns, name),
  },
})