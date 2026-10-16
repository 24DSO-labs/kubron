import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const limitRange = defineResource({
  id: 'limitrange',
  kind: 'LimitRange',
  view: 'limitranges',
  title: 'LimitRanges',
  namespaced: true,

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'items', label: 'Limits' },
    ageCol,
  ],

  properties: [
    { label: 'Limits', value: lr => lr.items },
  ],

  api: {
    list: ns => callApp('GetLimitRanges', ns),
    remove: (ns, name) => callApp('DeleteLimitRange', ns, name),
  },
})
