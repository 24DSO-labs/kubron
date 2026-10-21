import { defineResource } from '../../types'
import { callApp } from '../../wails'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoClusterInfo } from '../../../types/argocd'

const yesNo = (v: boolean) => (v ? 'Yes' : 'No')
const clusterLabel = (c: ArgoClusterInfo) => (c.inCluster ? 'in-cluster' : c.clusterName || c.name)

/**
 * Кластеры Argo CD (Secret с лейблом secret-type=cluster).
 * Только чтение и без YAML: config содержит токены и сертификаты, из него выводится лишь тип аутентификации.
 */
export const argoCluster = defineResource<ArgoClusterInfo>({
  id: 'argocluster',
  kind: 'Cluster',
  view: 'argo-clusters',
  title: 'Clusters',
  namespaced: true,
  editable: false,
  nav: { group: 'argocd-settings', label: 'Clusters' },

  autoRefresh: true,

  columns: [
    { ...flatNameCol, formatter: clusterLabel },
    flatNsCol,
    { key: 'server', label: 'Server' },
    { key: 'project', label: 'Project', formatter: c => c.project || '—' },
    { key: 'authType', label: 'Auth' },
    { key: 'namespaces', label: 'Namespaces', formatter: c => c.namespaces || 'all' },
    ageCol,
  ],

  properties: [
    { label: 'Secret', value: c => c.name },
    { label: 'Name', value: c => c.clusterName, hideEmpty: true },
    { label: 'Server', value: c => c.server },
    { label: 'Project', value: c => c.project, hideEmpty: true },
    { label: 'Authentication', value: c => c.authType },
    { label: 'Insecure TLS', value: c => yesNo(c.insecure) },
    { label: 'Namespaces', value: c => c.namespaces || 'all' },
    { label: 'Cluster Resources', value: c => yesNo(c.clusterResources) },
  ],

  api: {
    list: ns => callApp('GetArgoClusters', ns) as Promise<ArgoClusterInfo[]>,
  },
})
