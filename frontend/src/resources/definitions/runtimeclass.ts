// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import KeyValueSection from '../../components/resource/sections/KeyValueSection.vue'
import TolerationsSection from '../../components/resource/sections/TolerationsSection.vue'
import { h, defineComponent } from 'vue'
import { podsByClassTab } from '../podsByClass'

/** KeyValueSection принимает `data` пропом, а секции получают `resource`: тонкие обёртки над полями */
const mapSection = (title: string, field: 'overhead' | 'nodeSelector') =>
  defineComponent({
    name: `RuntimeClass${field}Section`,
    props: { resource: { type: Object, default: () => ({}) } },
    setup: props => () => h(KeyValueSection, { title, data: (props.resource as any)?.[field] }),
  })

export const runtimeClass = defineResource<k8s.RuntimeClassInfo>({
  id: 'runtimeclass',
  kind: 'RuntimeClass',
  view: 'runtimeclasses',
  title: 'RuntimeClasses',
  namespaced: false,
  nav: { group: 'config', label: 'Runtime Classes' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'handler', label: 'defs.columns.handler' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.handler', value: (r: any) => r.handler },
    { label: 'defs.props.finalizers', value: (r: any) => (r.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],
  sections: [mapSection('defs.runtimeClass.overhead', 'overhead'), mapSection('defs.columns.nodeSelector', 'nodeSelector'), TolerationsSection],

  tabs: [
    { id: 'pods', label: 'defs.tabs.pods', component: podsByClassTab('runtimeClassName') },
  ],

  api: {
    list: () => callApp('GetRuntimeClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'runtimeclass', '', name),
  },
})
