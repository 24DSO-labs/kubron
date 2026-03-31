<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <ContentHeader icon="" title="RoleBindings" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Namespace</th><th>Role</th><th>Age</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(rb, idx) in items" :key="idx" @click="$emit('select', rb)" class="clickable-row">
          <td class="resource-name">{{ rb.metadata?.name }}</td>
          <td class="ns-label">{{ rb.metadata?.namespace }}</td>
          <td>{{ rb.roleRef?.name }}</td>
          <td>{{ formatAge(rb.metadata?.creationTimestamp) }}</td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(rb, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет RoleBindings</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails">
        <span class="ctx-menu-arrow"></span>Show Details
      </div>
      <div class="ctx-menu-item" @click="handleEdit">
        <span class="ctx-menu-arrow"></span>Edit
      </div>
      <div class="ctx-menu-item" @click="handleDelete">
        <span class="ctx-menu-arrow"></span>Delete
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { RoleBinding } from '../../types'
import ContentHeader from '../ContentHeader.vue'
import { ref } from 'vue'

defineProps<{ items: RoleBinding[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', rb: RoleBinding): void
  (e: 'edit', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedRB = ref<RoleBinding | null>(null)

function formatAge(timestamp?: string): string {
  if (!timestamp) return '—'
  const date = new Date(timestamp)
  const now = new Date()
  const seconds = Math.floor((now.getTime() - date.getTime()) / 1000)
  if (seconds < 60) return `${seconds}s`
  if (seconds < 3600) return `${Math.floor(seconds / 60)}m`
  if (seconds < 86400) return `${Math.floor(seconds / 3600)}h`
  return `${Math.floor(seconds / 86400)}d`
}

function openContextMenu(rb: RoleBinding, event: MouseEvent) {
  selectedRB.value = rb
  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = 120

  let left = rect.left >= menuWidth ? rect.left - menuWidth + window.scrollX : rect.right + window.scrollX
  if (left + menuWidth > window.innerWidth + window.scrollX) left = window.innerWidth - menuWidth + window.scrollX

  let top = window.innerHeight - rect.bottom >= menuHeight ? rect.bottom + window.scrollY : rect.top - menuHeight + window.scrollY
  if (top < window.scrollY) top = window.scrollY

  menuStyle.value = { position: 'absolute', top: `${top}px`, left: `${left}px` }
  showContextMenu.value = true
}

function hideContextMenu() { showContextMenu.value = false }
function handleShowDetails() { if (selectedRB.value) emit('select', selectedRB.value); hideContextMenu() }
function handleEdit() { if (selectedRB.value) emit('edit', selectedRB.value.metadata?.namespace || '', selectedRB.value.metadata?.name || ''); hideContextMenu() }
function handleDelete() { if (selectedRB.value) emit('delete', selectedRB.value.metadata?.namespace || '', selectedRB.value.metadata?.name || ''); hideContextMenu() }
function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>

<style scoped>
.menu-overlay { position: fixed; top: 0; left: 0; width: 100%; height: 100%; z-index: 999; cursor: default; }
.cluster-context-menu { background: #1e1f35; border: 1px solid #3b3d5c; border-radius: 4px; box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3); z-index: 1000; min-width: 160px; }
.ctx-menu-item { padding: 6px 12px; color: #d4d4d4; cursor: pointer; font-size: 0.875rem; transition: background 0.15s; padding-left: 34px; text-align: left; position: relative; margin: -9px; }
.ctx-menu-item:hover { background: rgba(255, 255, 255, 0.1); color: white; }
.ctx-menu-item:first-child { border-radius: 3px 3px 0 0; }
.ctx-menu-item:last-child { border-radius: 0 0 3px 3px; }
.ctx-menu-arrow { position: absolute; left: 19px; color: #868d8a; font-size: 0.9rem; }
</style>
