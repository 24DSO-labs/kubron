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
  
  autoRefresh: true,icon: 'secret',

  columns: [
    flatNameCol,
    flatNsCol,
    { key: 'type', label: 'Type' },
    { key: 'dataCount', label: 'Data' },
    ageCol,
  ],

  properties: [
    { label: 'Type', value: s => s.type },
  ],
  sections: [SecretDataSection],

  api: {
    list: ns => callApp('GetSecrets', ns),
    remove: (ns, name) => callApp('DeleteSecret', ns, name),
  },
})
