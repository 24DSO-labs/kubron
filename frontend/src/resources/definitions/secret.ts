// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import SecretDataSection from '../../components/resource/sections/SecretDataSection.vue'

export const secret = defineResource<k8s.SecretInfo>({
  id: 'secret',
  kind: 'Secret',
  view: 'secrets',
  title: 'Secrets',
  namespaced: true,
  nav: { group: 'config', label: 'Secrets' },
  
  autoRefresh: true,
  // icon: 'secret',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (s: any) => s.namespace ? { id: 'namespace', ns: '', name: s.namespace } : null },
    { key: 'dataCount', label: 'defs.columns.keys' },
    { key: 'type', label: 'defs.columns.type' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.type', value: s => s.type },
    controlledByProp,
    managedByProp,
  ],

  tabs: [
    { id: 'data', label: 'defs.tabs.data', component: SecretDataSection },
  ],

  api: {
    list: ns => callApp('GetSecrets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'secret', ns, name),
  },
})
