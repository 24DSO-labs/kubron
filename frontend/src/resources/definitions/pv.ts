// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import { badgeClass, statusColor } from '../status'
import { expandAccessModes } from '../utils'
import PVMetricsSection from '../../components/resource/sections/PVMetricsSection.vue'
import PVProviderTab from '../../components/resource/sections/PVProviderTab.vue'
import PVClaimTab from '../../components/resource/sections/PVClaimTab.vue'

const scLink = (p: any) => (p.storageClass ? { id: 'storageclass', ns: '', name: p.storageClass } : null)
const claimLink = (p: any) =>
  p.claimName ? { id: 'persistentvolumeclaim', ns: p.claimNamespace, name: p.claimName } : null

export const persistentVolume = defineResource<k8s.PVInfo>({
  id: 'persistentvolume',
  kind: 'PersistentVolume',
  view: 'pv',
  title: 'PersistentVolumes',
  namespaced: false,
  nav: { group: 'storage', label: 'Persistent Volumes' },
  autoRefresh: true,

  // Как в Lens: Name, Storage Class, Capacity, Claim, Age, Status
  columns: [
    flatNameCol,
    { key: 'storageClass', label: 'defs.columns.storageClass', link: scLink },
    { key: 'capacity', label: 'defs.columns.capacity' },
    { key: 'claim', label: 'defs.columns.claim', link: claimLink },
    ageCol,
    { key: 'status', label: 'common.status', colorFn: p => statusColor(p.status) },
  ],

  headSections: [PVMetricsSection],

  // Name / Age / Created выводит шапка панели; Labels и Annotations — общие секции
  properties: [
    { label: 'defs.props.finalizers', value: (p: any) => (p.finalizers ?? []).join(', ') },
    { label: 'defs.columns.capacity', value: p => p.capacity },
    { label: 'defs.props.accessModes', value: (p: any) => expandAccessModes(p.accessModes) },
    { label: 'defs.columns.reclaimPolicy', value: p => p.reclaimPolicy },
    { label: 'defs.props.storageClassName', value: p => p.storageClass, link: scLink },
    { label: 'defs.props.volumeMode', value: (p: any) => p.volumeMode },
    
    { label: 'defs.columns.claim', value: p => p.claim, link: claimLink },
    { label: 'common.status', value: p => p.status, badge: p => badgeClass(p.status) },
    controlledByProp,
    managedByProp,
  ],
  tabs: [
    { id: 'provider', label: 'defs.tabs.provider', component: PVProviderTab },
    { id: 'claim', label: 'defs.tabs.claim', component: PVClaimTab },
  ],

  api: {
    list: () => callApp('GetPersistentVolumes'),
    remove: (_ns, name) => callApp('DeleteResource', 'persistentvolume', '', name),
  },
})