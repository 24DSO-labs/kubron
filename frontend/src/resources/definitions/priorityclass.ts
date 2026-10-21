import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import { badgeClass } from '../status'
import PriorityClassPodsTab from '../../components/resource/sections/PriorityClassPodsTab.vue'

const yesNo = (v: boolean) => (v ? 'True' : 'False')

export const priorityClass = defineResource<k8s.PriorityClassInfo>({
  id: 'priorityclass',
  kind: 'PriorityClass',
  view: 'priorityclasses',
  title: 'PriorityClasses',
  namespaced: false,
  nav: { group: 'config', label: 'Priority Classes' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'value', label: 'Value' },
    { key: 'globalDefault', label: 'Global Default', formatter: (p: any) => yesNo(p.globalDefault), colorFn: (p: any) => (p.globalDefault ? '#4ade80' : '#868d8a') },
    ageCol,
  ],

  properties: [
    { label: 'Value', value: (p: any) => p.value },
    { label: 'Global Default', value: (p: any) => yesNo(p.globalDefault), badge: (p: any) => (p.globalDefault ? badgeClass('Active') : 'rd-unk') },
    { label: 'Preemption Policy', value: (p: any) => p.preemptionPolicy },
    { label: 'Description', value: (p: any) => p.description },
    { label: 'Finalizers', value: (p: any) => (p.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],

  tabs: [
    { id: 'pods', label: 'Pods', component: PriorityClassPodsTab },
  ],

  api: {
    list: () => callApp('GetPriorityClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'priorityclass', '', name),
  },
})
