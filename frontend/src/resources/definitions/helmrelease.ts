import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol } from './common'
import { badgeClass, statusColor } from '../status'

export const helmRelease = defineResource<k8s.HelmReleaseInfo>({
  id: 'helmrelease',
  kind: 'Helm Release',
  view: 'helm-releases',
  title: 'Helm Releases',
  namespaced: false,
  nav: { group: 'helm', label: 'Releases' },
  
  autoRefresh: true,
  // icon: 'documentation',
  // helm-релиз — агрегат из secret-ов, не отдельный k8s-ресурс: без YAML и удаления
  editable: false,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (h: any) => h.namespace ? { id: 'namespace', ns: '', name: h.namespace } : null },
    { key: 'chartVersion', label: 'Chart Version' },
    { key: 'appVersion', label: 'App Version' },
    { key: 'revision', label: 'Revision' },
    { key: 'status', label: 'Status', colorFn: r => statusColor(r.status) },
    { key: 'updated', label: 'Updated' },
  ],

  properties: [
    { label: 'Status', value: r => r.status, badge: r => badgeClass(r.status) },
    { label: 'Chart', value: r => r.chart },
    { label: 'Chart Version', value: r => r.chartVersion },
    { label: 'App Version', value: r => r.appVersion },
    { label: 'Revision', value: r => r.revision },
    { label: 'Updated', value: r => r.updated },
    { label: 'Description', value: r => r.description },
  ],

  api: {
    list: () => callApp('GetHelmReleases'),
  },
})