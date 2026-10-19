import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import SecretDataSection from '../../components/resource/sections/SecretDataSection.vue'

export const secret = defineResource<k8s.SecretInfo>({
  id: 'secret',
  kind: 'Secret',
  view: 'secrets',
  title: 'Secrets',
  namespaced: true,
  nav: { group: 'config', label: 'Secrets' },
  
  autoRefresh: true,
  // icon: 'secret',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (s: any) => s.namespace ? { id: 'namespace', ns: '', name: s.namespace } : null },
    { key: 'dataCount', label: 'Keys' },
    { key: 'type', label: 'Type' },
    ageCol,
  ],

  properties: [
    { label: 'Type', value: s => s.type },
  ],

  tabs: [
    { id: 'data', label: 'Data', component: SecretDataSection },
  ],

  api: {
    list: ns => callApp('GetSecrets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'secret', ns, name),
  },
})
