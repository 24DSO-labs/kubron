import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const event = defineResource({
  id: 'event',
  kind: 'Event',
  view: 'events',
  title: 'Events',
  namespaced: true,
  nav: { label: 'Events' },

  columns: [
    flatNameCol,
    {
      key: 'type',
      label: 'Type',
      colorFn: e => statusColor(e.type),
    },
    { key: 'reason', label: 'Reason' },
    { key: 'object', label: 'Object' },
    flatNsCol,
    { key: 'message', label: 'Message' },
    { key: 'count', label: 'Count' },
    ageCol,
  ],

  properties: [
    { label: 'Type', value: e => e.type, badge: e => badgeClass(e.type) },
    { label: 'Reason', value: e => e.reason },
    { label: 'Object', value: e => e.object },
    { label: 'Message', value: e => e.message },
    { label: 'Count', value: e => e.count },
    { label: 'Source', value: e => e.source },
  ],

  // Events создаёт control plane: удалять вручную не нужно
  api: {
    list: ns => callApp('GetEvents', ns),
  },
})
