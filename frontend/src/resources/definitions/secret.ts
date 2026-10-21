import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import SecretDataSection from '../../components/resource/sections/SecretDataSection.vue'

/** "Kind/name" -> "Kind name" (имя отдельно, как в Lens); без "/" возвращаем как есть */
const splitRef = (ref?: string): [string, string] => {
  if (!ref) return ['', '']
  const i = ref.indexOf('/')
  return i < 0 ? [ref, ''] : [ref.slice(0, i), ref.slice(i + 1)]
}

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
    {
      // Владельцы секретов обычно CRD (ExternalSecret, SealedSecret, Certificate): приложение их не
      // открывает, поэтому показываем текстом
      label: 'Controlled By',
      value: (s: any) => {
        const [kind, name] = splitRef(s.controlledBy)
        return kind ? `${kind} ${name}`.trim() : null
      },
    },
    { label: 'Managed by', value: (s: any) => s.managedBy },
  ],

  tabs: [
    { id: 'data', label: 'Data', component: SecretDataSection },
  ],

  api: {
    list: ns => callApp('GetSecrets', ns),
    remove: (ns, name) => callApp('DeleteResource', 'secret', ns, name),
  },
})
