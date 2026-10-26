// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, shallowRef } from 'vue'
import { defineResource, type ColumnDef, type ResourceDefinition } from './types'
import { callApp } from './wails'
import { badgeClass, statusColor } from './status'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './definitions/common'
import { registerDynamicResource, clearDynamicResources } from './dynamicRegistry'
import ConditionsSection from '../components/resource/sections/ConditionsSection.vue'
import type { CRDInfo, CustomResourceInfo } from '../types/crd'

/** Список CRD активного кластера (для бокового меню) */
export const crds = shallowRef<CRDInfo[]>([])
/** Запрос перехода на view (App.vue подписан и вызывает onNavigate) */
export const navigationRequest = ref<string | null>(null)

export const crKey = (c: Pick<CRDInfo, 'group' | 'version' | 'plural'>) => `cr:${c.group}/${c.version}/${c.plural}`

export async function loadCustomResources(): Promise<void> {
  try {
    crds.value = ((await callApp('GetCRDs')) as unknown as CRDInfo[]) ?? []
  } catch {
    crds.value = [] // нет прав на CRD: раздел остаётся пустым
  }
}

export function resetCustomResources() {
  crds.value = []
  clearDynamicResources()
}

const STATUS_LIKE = /status|phase|state|health|sync|ready/i

/** Определение ресурса для CRD: те же список / панель / YAML / удаление, что у встроенных типов */
export function buildCustomResourceDef(crd: CRDInfo): ResourceDefinition<CustomResourceInfo> {
  const key = crKey(crd)
  const namespaced = crd.scope === 'Namespaced'
  const cell = (name: string) => (r: CustomResourceInfo) => r.cells?.[name] ?? ''

  const columns: ColumnDef[] = [
    flatNameCol,
    ...(namespaced
      ? [{ ...flatNsCol, link: (r: any) => (r.namespace ? { id: 'namespace', ns: '', name: r.namespace } : null) }]
      : []),
    ...crd.columns
      .filter(c => c.priority === 0)
      .map((c, i): ColumnDef => ({
        key: `c${i}`,
        label: c.name,
        formatter: (r: any) => cell(c.name)(r) || '—',
        colorFn: STATUS_LIKE.test(c.name) ? (r: any) => statusColor(cell(c.name)(r)) : undefined,
      })),
    ageCol,
  ]

  return defineResource<CustomResourceInfo>({
    id: key,
    kind: crd.kind,
    view: key,
    title: `${crd.kind} (${crd.group})`,
    namespaced,
    autoRefresh: true,
    columns,
    properties: [
      { label: 'API Version', value: r => r.apiVersion },
      ...crd.columns.map(c => ({
        label: c.name,
        value: (r: CustomResourceInfo) => cell(c.name)(r) || null,
        hideEmpty: true,
        badge: STATUS_LIKE.test(c.name) ? (r: CustomResourceInfo) => badgeClass(cell(c.name)(r)) : undefined,
      })),
      { label: 'Finalizers', value: r => (r.finalizers ?? []).join(', ') || null, hideEmpty: true },
      controlledByProp,
      managedByProp,
    ],
    sections: [ConditionsSection],
    api: {
      list: ns => callApp('GetCustomResources', crd.group, crd.version, crd.plural, ns) as Promise<CustomResourceInfo[]>,
      // ключ "cr:group/version/plural" проверяется на Go через CRD
      remove: (ns, name) => callApp('DeleteResource', key, ns, name),
    },
  })
}

/** Регистрирует определение и возвращает view, на который можно перейти */
export function ensureCustomResourceView(crd: CRDInfo): string {
  registerDynamicResource(buildCustomResourceDef(crd))
  return crKey(crd)
}

export function browseCustomResource(crd: CRDInfo) {
  navigationRequest.value = ensureCustomResourceView(crd)
}