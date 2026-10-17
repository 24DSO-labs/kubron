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
  // агрегация установленных релизов по chart-у: без YAML и удаления
  editable: false,

  columns: [
    flatNameCol,
    { key: 'version', label: 'Version' },
    { key: 'appVersion', label: 'App Version' },
    { key: 'releases', label: 'Releases' },
    { key: 'namespaces', label: 'Namespaces' },
    { key: 'updated', label: 'Updated' },
  ],

  properties: [
    { label: 'Chart', value: c => c.name },
    { label: 'Version', value: c => c.version },
    { label: 'App Version', value: c => c.appVersion },
    { label: 'Releases', value: c => c.releases },
    { label: 'Namespaces', value: c => c.namespaces },
    { label: 'Updated', value: c => c.updated },
    { label: 'Description', value: c => c.description },
  ],

  api: {
    list: () => callApp('GetHelmCharts'),
  },
})