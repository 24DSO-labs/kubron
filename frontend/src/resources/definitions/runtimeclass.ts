import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import KeyValueSection from '../../components/resource/sections/KeyValueSection.vue'
import TolerationsSection from '../../components/resource/sections/TolerationsSection.vue'
import { h, defineComponent } from 'vue'

/** KeyValueSection принимает `data` пропом, а секции получают `resource`: тонкие обёртки над полями */
const mapSection = (title: string, field: 'overhead' | 'nodeSelector') =>
  defineComponent({
    name: `RuntimeClass${field}Section`,
    props: { resource: { type: Object, default: () => ({}) } },
    setup: props => () => h(KeyValueSection, { title, data: (props.resource as any)?.[field] }),
  })

export const runtimeClass = defineResource<k8s.RuntimeClassInfo>({
  id: 'runtimeclass',
  kind: 'RuntimeClass',
  view: 'runtimeclasses',
  title: 'RuntimeClasses',
  namespaced: false,
  nav: { group: 'config', label: 'Runtime Classes' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'handler', label: 'Handler' },
    ageCol,
  ],

  properties: [
    { label: 'Handler', value: (r: any) => r.handler },
    { label: 'Finalizers', value: (r: any) => (r.finalizers ?? []).join(', ') },
  ],
  sections: [mapSection('Overhead', 'overhead'), mapSection('Node Selector', 'nodeSelector'), TolerationsSection],

  api: {
    list: () => callApp('GetRuntimeClasses'),
    remove: (_ns, name) => callApp('DeleteResource', 'runtimeclass', '', name),
  },
})
