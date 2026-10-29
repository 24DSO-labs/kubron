// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import { t } from '../../i18n'

export const lease = defineResource<k8s.LeaseInfo>({
  id: 'lease',
  kind: 'Lease',
  view: 'leases',
  title: 'Leases',
  namespaced: true,
  nav: { group: 'config', label: 'Leases' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (l: any) => l.namespace ? { id: 'namespace', ns: '', name: l.namespace } : null },
    { key: 'holder', label: 'defs.columns.holder' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.holder', value: (l: any) => l.holder },
    { label: 'defs.props.preferredHolder', value: (l: any) => l.preferredHolder },
    { label: 'defs.props.strategy', value: (l: any) => l.strategy },
    { label: 'defs.props.leaseDuration', value: (l: any) => (l.leaseDurationSeconds ? t('defs.lease.seconds', { n: l.leaseDurationSeconds }) : '') },
    { label: 'defs.props.leaseTransitions', value: (l: any) => l.leaseTransitions },
    { label: 'defs.props.acquireTime', value: (l: any) => l.acquireTime },
    { label: 'defs.props.renewTime', value: (l: any) => l.renewTime },
    { label: 'defs.props.finalizers', value: (l: any) => (l.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],

  api: {
    list: ns => callApp('GetLeases', ns),
    remove: (ns, name) => callApp('DeleteResource', 'lease', ns, name),
  },
})
