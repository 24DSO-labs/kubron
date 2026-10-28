// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../../types'
import { t } from '../../../i18n'
import { callApp } from '../../wails'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoClusterInfo } from '../../../types/argocd'

const yesNo = (v: boolean) => (v ? t('common.yes') : t('common.no'))
const allNamespaces = (c: ArgoClusterInfo) => c.namespaces || t('argo.allNamespaces')
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
    { key: 'server', label: 'argo.columns.server' },
    { key: 'project', label: 'argo.columns.project', formatter: c => c.project || '—' },
    { key: 'authType', label: 'argo.columns.auth' },
    { key: 'namespaces', label: 'defs.columns.namespaces', formatter: allNamespaces },
    ageCol,
  ],

  properties: [
    { label: 'argo.props.secret', value: c => c.name },
    { label: 'common.name', value: c => c.clusterName, hideEmpty: true },
    { label: 'argo.columns.server', value: c => c.server },
    { label: 'argo.columns.project', value: c => c.project, hideEmpty: true },
    { label: 'argo.props.authentication', value: c => c.authType },
    { label: 'argo.props.insecureTls', value: c => yesNo(c.insecure) },
    { label: 'defs.columns.namespaces', value: allNamespaces },
    { label: 'argo.props.clusterResources', value: c => yesNo(c.clusterResources) },
  ],

  api: {
    list: ns => callApp('GetArgoClusters', ns) as Promise<ArgoClusterInfo[]>,
  },
})
