import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import { browseCustomResource, crds } from '../customResources'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import type { CRDInfo } from '../../types/crd'
import { Perm } from '../../types/auth'

export const customResourceDefinition = defineResource<CRDInfo>({
  id: 'customresourcedefinition',
  kind: 'CustomResourceDefinition',
  view: 'crds',
  title: 'Definitions',
  namespaced: false,
  autoRefresh: true,

  columns: [
    flatNameCol,
    { key: 'group', label: 'Group' },
    { key: 'kind', label: 'Kind' },
    { key: 'scope', label: 'Scope' },
    { key: 'version', label: 'Version' },
    ageCol,
  ],

  properties: [
    { label: 'Group', value: c => c.group },
    { label: 'Kind', value: c => c.kind },
    { label: 'Plural', value: c => c.plural },
    { label: 'Scope', value: c => c.scope },
    { label: 'Preferred Version', value: c => c.version },
    { label: 'Served Versions', value: c => c.versions.join(', ') },
    { label: 'Short Names', value: c => c.shortNames.join(', ') || null, hideEmpty: true },
    { label: 'Categories', value: c => c.categories.join(', ') || null, hideEmpty: true },
    controlledByProp,
    managedByProp,
  ],
  sections: [ConditionsSection],

  actions: [
    {
      id: 'browse',
      label: 'Browse resources',
      icon: 'view',
      permission: Perm.Read,
      run: c => browseCustomResource(c as CRDInfo),
    },
  ],

  // Список обновляет и боковое меню
  onLoaded: items => {
    crds.value = [...items] as CRDInfo[]
  },

  api: {
    list: () => callApp('GetCRDs') as unknown as Promise<CRDInfo[]>,
  },
})