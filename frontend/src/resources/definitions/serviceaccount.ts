// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, nsCol, ageCol, controlledByProp, managedByProp } from './common'

export const serviceAccount = defineResource({
  id: 'serviceaccount',
  kind: 'ServiceAccount',
  view: 'serviceaccounts',
  title: 'Service Accounts',
  // icon: 'service-account',
  namespaced: true,
  nav: { group: 'access', label: 'Service Accounts' },
  columns: [nameCol, { ...nsCol, link: (r: any) => r.metadata?.namespace ? { id: 'namespace', ns: '', name: r.metadata.namespace } : null }, ageCol],
  // TODO: сверить поля с ServiceAccountDetailPanel (secrets / imagePullSecrets)
  properties: [
    { label: 'Secrets', value: r => r.secrets?.length ?? 0 },
    { label: 'Image pull secrets', value: r => r.imagePullSecrets?.length ?? 0 },
    controlledByProp,
    managedByProp,
  ],
  api: {
    list: ns => callApp('GetServiceAccounts', ns),
    get: (ns, n) => callApp('GetServiceAccountDetail', ns, n),
    remove: (ns, n) => callApp('DeleteResource', 'serviceaccount', ns, n),
  },
})
