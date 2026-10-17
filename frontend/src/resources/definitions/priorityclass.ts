import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const priorityClass = defineResource<k8s.PriorityClassInfo>({
  id: 'priorityclass',
  kind: 'PriorityClass',
  view: 'priorityclasses',
  title: 'PriorityClasses',
  namespaced: false,
  nav: { group: 'config', label: 'Priority Classes' },

  columns: [
    flatNameCol,
    { key: 'value', label: 'Value' },
    ageCol,
  ],

  properties: [
    { label: 'Value', value: p => p.value },
  ],

  api: {
    list: () => callApp('GetPriorityClasses'),
    remove: (_ns, name) => callApp('DeletePriorityClass', name),
  },
})
