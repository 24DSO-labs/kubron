import type { ResourceDefinition } from './types'

/** Определения, созданные на лету (custom resources). Живут до смены кластера */
export const dynamicById = new Map<string, ResourceDefinition>()
export const dynamicByView = new Map<string, ResourceDefinition>()

export function registerDynamicResource(def: ResourceDefinition) {
  dynamicById.set(def.id.toLowerCase(), def)
  dynamicByView.set(def.view, def)
}

export function clearDynamicResources() {
  dynamicById.clear()
  dynamicByView.clear()
}