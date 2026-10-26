// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue'
import HelmContentTab from '../components/resource/sections/HelmContentTab.vue'

/** TabDef.component получает только `resource`: вид содержимого фиксируем здесь */
export const helmContentTab = (kind: 'values' | 'notes' | 'manifest') =>
  defineComponent({
    name: `HelmContentTab_${kind}`,
    props: { resource: { type: Object, default: () => ({}) } },
    setup: props => () => h(HelmContentTab, { resource: props.resource, kind }),
  })