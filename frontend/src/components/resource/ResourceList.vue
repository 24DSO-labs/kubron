<template>
  <div v-if="initialLoading" class="loading"> Загрузка...</div>
  <ResourceTable
    v-else
    :items="items"
    :search="search"
    :title="def.title"
    :icon="def.icon"
    :empty-message="`Нет ${def.title}`"
    :columns="def.columns"
    :menu-for="menuFor"
    :get-item-key="itemKey"
    :refreshing="refreshing"
    :updated-at="updatedAt ? updatedAt.toLocaleTimeString() : undefined"
    @refresh="reload"
    @search="(v: string) => (search = v)"
    @select="(item: any) => open(item)"
    @open="onOpen"
    @action="onAction"
  />
</template>

<script lang="ts" setup>
import ResourceTable from './ResourceTable.vue'
import type { MenuItemDef, ResourceDefinition } from '../../resources/types'
import { useResourceList } from '../../resources/store'
import { resourceActions } from '../../resources/actions'
import { actionAllowed, canDelete, canEditYaml } from '../../resources/access'
import { containersOf, itemName, itemNs } from '../../resources/utils'

const props = defineProps<{ def: ResourceDefinition }>()

const { items, initialLoading, reload, refreshing, updatedAt, search } = useResourceList(() => props.def)

/**
 * Меню строится для конкретной строки. Действие с perContainer получает подменю контейнеров,
 * если их больше одного; с одним контейнером запускается сразу (как кнопка в панели деталей).
 */
function menuFor(item: any): MenuItemDef[] {
  const containers = containersOf(item)
  return [
    { label: 'Show Details', action: 'select' },
    ...(props.def.actions ?? []).filter(actionAllowed).map(a => ({
      label: a.label,
      action: a.id,
      danger: a.danger,
      submenu:
        a.perContainer && containers.length > 1
          ? containers.map(c => ({ label: c.name, arg: c.name }))
          : undefined,
    })),
    // Без права на запись YAML открывается только для просмотра
    ...(props.def.editable !== false ? [{ label: canEditYaml() ? 'Edit' : 'View YAML', action: 'edit' }] : []),
    ...(canDelete(props.def) ? [{ label: 'Delete', action: 'delete', danger: true }] : []),
  ]
}

// Ключ строки: у элементов с metadata.name простой item.name был бы пустым
const itemKey = (item: any) => `${itemNs(item)}/${itemName(item)}`

function open(item: any) {
  resourceActions.open(props.def.id, itemNs(item), itemName(item), item)
}

function onOpen(data: { id: string; ns: string; name: string }) {
  resourceActions.open(data.id, data.ns, data.name)
}

function onAction(action: string, item: any, container?: string) {
  const ns = itemNs(item)
  const name = itemName(item)

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
      return a.run(item, resourceActions, container)
    }
  }
}
</script>

<style scoped>
.loading { padding: 24px; color: #868d8a; }
:deep(.resource-name) { font-weight: 500; }
:deep(.ns-label) { color: #868d8a; }
</style>
