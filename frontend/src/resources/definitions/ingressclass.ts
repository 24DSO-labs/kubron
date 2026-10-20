import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import { badgeClass } from '../status'
import IngressClassUsedBySection from '../../components/resource/sections/IngressClassUsedBySection.vue'

const yesNo = (v: boolean) => (v ? 'True' : 'False')

export const ingressClass = defineResource<k8s.IngressClassInfo>({
  id: 'ingressclass',
  kind: 'IngressClass',
  view: 'ingressclasses',
  title: 'IngressClasses',
  namespaced: false,
  nav: { group: 'network', label: 'Ingress Classes' },
  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'controller', label: 'Controller' },
    { key: 'paramsKind', label: 'Parameters', formatter: (c: any) => (c.paramsKind ? `${c.paramsKind}/${c.paramsName}` : '—') },
    { key: 'default', label: 'Default', formatter: (c: any) => yesNo(c.default), colorFn: (c: any) => (c.default ? '#4ade80' : '#868d8a') },
    {
      key: 'usedBy',
      label: 'Used by',
      formatter: (c: any) => (c.usedBy ?? []).map((u: any) => u.name).join(', ') || '—',
    },
    ageCol,
  ],

  properties: [
    { label: 'Controller', value: (c: any) => c.controller },
    { label: 'Default', value: (c: any) => yesNo(c.default), badge: (c: any) => (c.default ? badgeClass('Active') : 'rd-unk') },
    { label: 'Parameters Kind', value: (c: any) => c.paramsKind },
    { label: 'Parameters Name', value: (c: any) => c.paramsName },
    { label: 'Parameters API Group', value: (c: any) => c.paramsApiGroup },
    { label: 'Parameters Scope', value: (c: any) => c.paramsScope },
    { label: 'Parameters Namespace', value: (c: any) => c.paramsNamespace },
  ],
  sections: [IngressClassUsedBySection],
  api: {
    list: () => callApp('GetIngressClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'ingressclass', '', name),
  },
})