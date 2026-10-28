// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const event = defineResource<k8s.EventInfo>({
  id: 'event',
  kind: 'Event',
  view: 'events',
  title: 'Events',
  namespaced: true,
  nav: { label: 'Events' },

  autoRefresh: true,
  // icon: 'events',

  // События создаёт control plane: YAML не редактируется, пункты Edit/View YAML скрыты
  editable: false,

  columns: [
    flatNameCol,
    {
      key: 'type',
      label: 'defs.columns.type',
      colorFn: e => statusColor(e.type),
    },
    { key: 'reason', label: 'defs.columns.reason' },
    { key: 'object', label: 'defs.columns.object' },
    { ...flatNsCol, link: (e: any) => e.namespace ? { id: 'namespace', ns: '', name: e.namespace } : null },
    { key: 'message', label: 'defs.columns.message' },
    { key: 'count', label: 'defs.columns.count' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.type', value: e => e.type, badge: e => badgeClass(e.type) },
    { label: 'defs.columns.reason', value: e => e.reason },
    { label: 'defs.columns.object', value: e => e.object },
    { label: 'defs.columns.message', value: e => e.message },
    { label: 'defs.columns.count', value: e => e.count },
    { label: 'defs.props.source', value: e => e.source },
  ],

  // Events создаёт control plane: удалять вручную не нужно
  api: {
    list: ns => callApp('GetEvents', ns),
  },
})