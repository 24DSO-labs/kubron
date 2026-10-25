<template>
  <div class="es-tab">
    <div v-if="!endpointSlices.length" class="es-empty">No Endpoint Slices</div>
    <table v-else class="es-table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Endpoints</th>
          <th>Ports</th>
          <th>Address Type</th>
          <th>Age</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="es in endpointSlices" :key="es.name" class="es-row">
          <td class="es-name-cell">{{ es.name }}</td>
          <td>{{ es.endpoints || '—' }}</td>
          <td>{{ es.ports?.join(', ') || '—' }}</td>
          <td>{{ es.addressType || '—' }}</td>
          <td>{{ es.age || '—' }}</td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openMenu(es, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>

    <!-- Overlay -->
    <div v-if="menu" class="menu-overlay" @click="hideMenu" @contextmenu.prevent="hideMenu" />

    <!-- Context menu -->
    <div v-if="menu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="showDetails(menu.item)">Show Details</div>
      <div class="ctx-menu-item" @click="editYaml(menu.item)">Edit</div>
      <div class="ctx-menu-item ctx-menu-danger" @click="remove(menu.item)">Delete</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { globalTabbedPanel } from '../../../composables/useTabbedPanel'
import { useNotifications } from '../../../composables/useNotifications'
import { askConfirm } from '../../../resources/confirm'
import { can } from '../../../composables/useAuth'
import { Perm } from '../../../types/auth'
import { callApp } from '../../../platform/api'

const props = defineProps<{ resource: any }>()
const endpointSlices = computed(() => props.resource?.endpointSlices ?? [])

const notifications = useNotifications()

/* ─── Context menu ─── */
const menu = ref<{ item: any; x: number; y: number } | null>(null)
const MENU_WIDTH = 160

const menuStyle = computed(() => ({
  position: 'fixed' as const,
  left: `${menu.value?.x ?? 0}px`,
  top: `${menu.value?.y ?? 0}px`,
  zIndex: 50,
}))

function openMenu(item: any, event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = rect.left >= MENU_WIDTH ? rect.left - MENU_WIDTH : rect.right
  menu.value = { item, x, y: rect.bottom }
}

function hideMenu() {
  menu.value = null
}

/* ─── Actions ─── */
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void }>()

function showDetails(item: any) {
  hideMenu()
  emit('open', { id: 'endpointslice', ns: item.namespace, name: item.name, item })
}

async function editYaml(item: any) {
  hideMenu()
  if (!can(Perm.Write)) {
    notifications.notifyError('Insufficient permissions to edit')
    return
  }
  try {
    const yaml = await callApp('GetResourceYAML', item.namespace, 'endpointslice', item.name, true)
    globalTabbedPanel.openYamlTab('endpointslice', item.name, item.namespace, yaml)
  } catch (e: any) {
    notifications.notifyError(`Failed to load YAML: ${e?.message || String(e)}`)
  }
}

async function remove(item: any) {
  hideMenu()
  if (!can(Perm.Write)) {
    notifications.notifyError('Insufficient permissions to delete')
    return
  }
  if (!(await askConfirm(`Delete EndpointSlice "${item.name}"?`))) return
  try {
    await callApp('DeleteResource', 'endpointslice', item.namespace, item.name)
    notifications.notifySuccess(`EndpointSlice "${item.name}" deleted`)
  } catch (e: any) {
    notifications.notifyError(`Failed to delete: ${e?.message || String(e)}`)
  }
}
</script>

<style scoped>
.es-tab { padding: 12px 16px; }
.es-empty { color: #868d8a; font-size: 0.875rem; padding: 16px 0; }

.es-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.es-table th {
  text-align: left;
  padding: 8px 12px;
  color: #9ca0b8;
  font-weight: 500;
  font-size: 0.8rem;
  text-transform: uppercase;
  letter-spacing: 0.04em;
  border-bottom: 1px solid #2a2b40;
}

.es-table td {
  padding: 10px 12px;
  border-bottom: 1px solid #252640;
  color: #c0c2d8;
  vertical-align: middle;
}

.es-row:hover td {
  background: rgba(255, 255, 255, 0.02);
}

.es-name-cell {
  font-weight: 500;
  color: #d4d4d4;
}

.btn-icon {
  background: transparent;
  border: none;
  color: #d4d4d4;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background 0.15s;
}
.btn-icon:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  z-index: 40;
}
</style>
