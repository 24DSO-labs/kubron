// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineComponent, h } from 'vue'
import PodsByClassTab from '../components/resource/sections/PodsByClassTab.vue'

/** Вкладка Pods для PriorityClass / RuntimeClass: TabDef.component получает только `resource`, поле фиксируем здесь */
export const podsByClassTab = (field: 'priorityClassName' | 'runtimeClassName') =>
  defineComponent({
    name: `PodsBy${field}Tab`,
    props: { resource: { type: Object, default: () => ({}) } },
    emits: ['open'],
    setup: (props, { emit }) => () =>
      h(PodsByClassTab, { resource: props.resource, field, onOpen: (p: any) => emit('open', p) }),
  })
