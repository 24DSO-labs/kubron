// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol } from './common'

export const helmChart = defineResource<k8s.HelmChartInfo>({
  id: 'helmchart',
  kind: 'Helm Chart',
  view: 'helm-charts',
  title: 'Helm Charts',
  namespaced: false,
  nav: { group: 'helm', label: 'Charts' },
  
  autoRefresh: true,
  // icon: 'documentation',
  // агрегация установленных релизов по chart-у: без YAML и удаления
  editable: false,

  columns: [
    flatNameCol,
    { key: 'version', label: 'defs.columns.version' },
    { key: 'appVersion', label: 'defs.columns.appVersion' },
    { key: 'releases', label: 'defs.columns.releases' },
    { key: 'namespaces', label: 'defs.columns.namespaces' },
    { key: 'updated', label: 'defs.columns.updated' },
  ],

  properties: [
    { label: 'defs.props.chart', value: c => c.name },
    { label: 'defs.columns.version', value: c => c.version },
    { label: 'defs.columns.appVersion', value: c => c.appVersion },
    { label: 'defs.columns.releases', value: c => c.releases },
    { label: 'defs.columns.namespaces', value: c => c.namespaces },
    { label: 'defs.columns.updated', value: c => c.updated },
    { label: 'defs.props.description', value: c => c.description },
  ],

  api: {
    list: () => callApp('GetHelmCharts'),
  },
})