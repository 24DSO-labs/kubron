// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

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
    { key: 'group', label: 'defs.columns.group' },
    { key: 'kind', label: 'defs.columns.kind' },
    { key: 'scope', label: 'defs.columns.scope' },
    { key: 'version', label: 'defs.columns.version' },
    ageCol,
  ],

  properties: [
    { label: 'defs.columns.group', value: c => c.group },
    { label: 'defs.columns.kind', value: c => c.kind },
    { label: 'defs.props.plural', value: c => c.plural },
    { label: 'defs.columns.scope', value: c => c.scope },
    { label: 'defs.props.preferredVersion', value: c => c.version },
    { label: 'defs.props.servedVersions', value: c => c.versions.join(', ') },
    { label: 'defs.props.shortNames', value: c => c.shortNames.join(', ') || null, hideEmpty: true },
    { label: 'defs.props.categories', value: c => c.categories.join(', ') || null, hideEmpty: true },
    controlledByProp,
    managedByProp,
  ],
  sections: [ConditionsSection],

  actions: [
    {
      id: 'browse',
      label: 'defs.menu.browseResources',
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