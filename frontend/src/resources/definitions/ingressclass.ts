import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'

export const ingressClass = defineResource({
  id: 'ingressclass',
  kind: 'IngressClass',
  view: 'ingressclasses',
  title: 'IngressClasses',
  namespaced: false,
  nav: { group: 'network', label: 'Ingress Classes' },

  columns: [
    flatNameCol,
    { key: 'controller', label: 'Controller' },
    ageCol,
  ],

  properties: [
    { label: 'Controller', value: ic => ic.controller },
  ],

  api: {
    list: () => callApp('GetIngressClasses'),
    remove: (_ns, name) => callApp('DeleteIngressClass', name),
  },
})
