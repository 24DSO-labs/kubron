<template>
  <div v-if="initialLoading" class="loading">⏳ Загрузка...</div>
  <K8sResourceListBase
    v-else
    :items="items"
    :search="search"
    :resource-title="def.title"
    :empty-message="`Нет ${def.title}`"
    :columns="def.columns"
    :context-menu-items="menu"
    :get-item-key="itemKey"
    @refresh="reload"
    @search="(v: string) => (search = v)"
    @select="(item: any) => open(item)"
    @context-action="onContextAction"
    @open-logs="onOpenLogs"
  />
</template>

<script lang="ts" setup>
import { computed } from 'vue'
// TODO: когда будет готова новая K8sResourceListBase (не _old), поменять только этот импорт
import K8sResourceListBase from '../views/K8sResourceListBase_old.vue'
import type { MenuItemDef, ResourceDefinition } from '../../resources/types'
import { useResourceList } from '../../resources/store'
import { resourceActions } from '../../resources/actions'
import { itemName, itemNs } from '../../resources/utils'

const props = defineProps<{ def: ResourceDefinition }>()

const { items, initialLoading, reload, search } = useResourceList(() => props.def)

const menu = computed<MenuItemDef[]>(() => [
  { label: 'Show Details', action: 'select' },
  ...(props.def.actions ?? []).map(a => ({ label: a.label, action: a.id, danger: a.danger })),
  ...(props.def.editable !== false ? [{ label: 'Edit', action: 'edit' }] : []),
  ...(props.def.api.remove ? [{ label: 'Delete', action: 'delete', danger: true }] : []),
])

// Базовый список строит ключ из item.name; для элементов с metadata.name он был бы одинаковым у всех строк
const itemKey = (item: any) => `${itemNs(item)}/${itemName(item)}`

function open(item: any) {
  resourceActions.open(props.def.id, itemNs(item), itemName(item), item)
}

function onContextAction(action: string, item: any) {
  const ns = itemNs(item)
  const name = itemName(item)

  // Per-container actions от старого базового компонента: attach:<container>, shell:<container>
  if (action.startsWith('attachPod:') || action.startsWith('attach:')) {
    const container = action.split(':')[1]
    return resourceActions.openAttach(ns, name, container)
  }
  if (action.startsWith('shell:')) {
    const container = action.split(':')[1]
    return resourceActions.openShell(ns, name, container)
  }

  switch (action) {
    case 'select':
      return open(item)
    case 'edit':
      return resourceActions.edit(props.def.id, ns, name)
    case 'delete':
      return resourceActions.requestDelete(props.def.id, ns, name)
    default: {
      const a = props.def.actions?.find(x => x.id === action)
      if (!a) return
      if (a.when && !a.when(item)) {
        return resourceActions.notifyError(`"${a.label}" недоступно для ${name}`)
      }
      return a.run(item, resourceActions)
    }
  }
}

function onOpenLogs(payload: { resource: any; selectedContainer: string; containers: { name: string }[] }) {
  resourceActions.openLogs(
    payload.resource.namespace,
    payload.resource.name,
    payload.containers,
    payload.selectedContainer,
  )
}
</script>

<style scoped>
.loading { padding: 24px; color: #868d8a; }
:deep(.resource-name) { font-weight: 500; }
:deep(.ns-label) { color: #868d8a; }
</style>
