// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../../types'
import { t } from '../../../i18n'
import { callApp } from '../../wails'
import { flatNameCol, flatNsCol, ageCol } from '../common'
import type { ArgoRepositoryInfo } from '../../../types/argocd'
import { shortRepo } from '../../argocd'

const yesNo = (v: boolean) => (v ? t('common.yes') : t('common.no'))

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
    { key: 'url', label: 'argo.columns.url', formatter: r => shortRepo(r.url) || '—' },
    { key: 'type', label: 'defs.columns.type' },
    { key: 'project', label: 'argo.columns.project', formatter: r => r.project || '—' },
    { key: 'authType', label: 'argo.columns.auth' },
    { key: 'insecure', label: 'argo.columns.insecure', formatter: r => yesNo(r.insecure) },
    ageCol,
  ],

  properties: [
    { label: 'argo.props.secret', value: r => r.name },
    { label: 'argo.columns.url', value: r => r.url },
    { label: 'defs.columns.type', value: r => r.type },
    { label: 'argo.columns.project', value: r => r.project, hideEmpty: true },
    { label: 'argo.props.authentication', value: r => r.authType },
    { label: 'argo.columns.insecure', value: r => yesNo(r.insecure) },
    { label: 'Git LFS', value: r => (r.enableLfs ? t('common.enabled') : null), hideEmpty: true },
    { label: 'OCI', value: r => (r.enableOCI ? t('common.enabled') : null), hideEmpty: true },
    { label: 'argo.props.forceBasicAuth', value: r => (r.forceHttpBasicAuth ? t('common.yes') : null), hideEmpty: true },
    { label: 'argo.props.proxy', value: r => (r.proxy ? t('argo.props.configured') : null), hideEmpty: true },
  ],

  api: {
    list: ns => callApp('GetArgoRepositories', ns) as Promise<ArgoRepositoryInfo[]>,
  },
})
