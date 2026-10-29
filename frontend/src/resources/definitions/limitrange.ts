// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import LimitRangeLimitsSection from '../../components/resource/sections/LimitRangeLimitsSection.vue'

export const limitRange = defineResource<k8s.LimitRangeInfo>({
  id: 'limitrange',
  kind: 'LimitRange',
  view: 'limitranges',
  title: 'LimitRanges',
  namespaced: true,
  nav: { group: 'config', label: 'Limit Ranges' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (lr: any) => lr.namespace ? { id: 'namespace', ns: '', name: lr.namespace } : null },
    { key: 'items', label: 'defs.columns.limits' },
    ageCol,
  ],

  properties: [
    { label: 'defs.props.finalizers', value: (lr: any) => (lr.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],
  sections: [LimitRangeLimitsSection],

  api: {
    list: ns => callApp('GetLimitRanges', ns),
    remove: (ns, name) => callApp('DeleteResource', 'limitrange', ns, name),
  },
})
