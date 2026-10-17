import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const runtimeClass = defineResource<k8s.RuntimeClassInfo>({
  id: 'runtimeclass',
  kind: 'RuntimeClass',
  view: 'runtimeclasses',
  title: 'RuntimeClasses',
  namespaced: false,
  nav: { group: 'config', label: 'Runtime Classes' },
  icon: 'configmap',

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
