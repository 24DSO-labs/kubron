// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../../types'
import { callApp } from '../../wails'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoRepositoryInfo } from '../../../types/argocd'
import { shortRepo } from '../../argocd'

const yesNo = (v: boolean) => (v ? 'Yes' : 'No')

/**
 * Репозитории Argo CD (Secret с лейблом secret-type=repository).
 * Только чтение и без YAML: Secret содержит пароли и ключи, бэкенд отдаёт санитизированные поля.
 */
export const argoRepository = defineResource<ArgoRepositoryInfo>({
  id: 'argorepository',
  kind: 'Repository',
  view: 'argo-repositories',
  title: 'Repositories',
  namespaced: true,
  editable: false,
  nav: { group: 'argocd-settings', label: 'Repositories' },

  autoRefresh: true,

  columns: [
    { ...flatNameCol, formatter: r => r.repoName || r.name },
    flatNsCol,
    { key: 'url', label: 'URL', formatter: r => shortRepo(r.url) || '—' },
    { key: 'type', label: 'Type' },
    { key: 'project', label: 'Project', formatter: r => r.project || '—' },
    { key: 'authType', label: 'Auth' },
    { key: 'insecure', label: 'Insecure', formatter: r => yesNo(r.insecure) },
    ageCol,
  ],

  properties: [
    { label: 'Secret', value: r => r.name },
    { label: 'URL', value: r => r.url },
    { label: 'Type', value: r => r.type },
    { label: 'Project', value: r => r.project, hideEmpty: true },
    { label: 'Authentication', value: r => r.authType },
    { label: 'Insecure', value: r => yesNo(r.insecure) },
    { label: 'Git LFS', value: r => (r.enableLfs ? 'Enabled' : null), hideEmpty: true },
    { label: 'OCI', value: r => (r.enableOCI ? 'Enabled' : null), hideEmpty: true },
    { label: 'Force HTTP Basic Auth', value: r => (r.forceHttpBasicAuth ? 'Yes' : null), hideEmpty: true },
    { label: 'Proxy', value: r => (r.proxy ? 'Configured' : null), hideEmpty: true },
  ],

  api: {
    list: ns => callApp('GetArgoRepositories', ns) as Promise<ArgoRepositoryInfo[]>,
  },
})
