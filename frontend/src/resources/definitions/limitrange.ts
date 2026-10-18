import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const limitRange = defineResource<k8s.LimitRangeInfo>({
  id: 'limitrange',
  kind: 'LimitRange',
  view: 'limitranges',
  title: 'LimitRanges',
  namespaced: true,
  nav: { group: 'config', label: 'Limit Ranges' },
  
  autoRefresh: true,icon: 'configmap',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (lr: any) => lr.namespace ? { id: 'namespace', ns: '', name: lr.namespace } : null },
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
