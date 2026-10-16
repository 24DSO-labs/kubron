import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const runtimeClass = defineResource({
  id: 'runtimeclass',
  kind: 'RuntimeClass',
  view: 'runtimeclasses',
  title: 'RuntimeClasses',
  namespaced: false,

  columns: [
    flatNameCol,
    { key: 'handler', label: 'Handler' },
    ageCol,
  ],

  properties: [
    { label: 'Handler', value: r => r.handler },
  ],

  api: {
    list: () => callApp('GetRuntimeClasses'),
    remove: (_ns, name) => callApp('DeleteRuntimeClass', name),
  },
})
