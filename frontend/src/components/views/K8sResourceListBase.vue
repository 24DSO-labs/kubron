<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <ContentHeader icon="" :title="resourceTitle" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th v-for="col in columns" :key="col.key">{{ col.label }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="getItemKey(item)" @click="handleRowClick(item)" class="clickable-row">
          <td v-for="col in columns" :key="col.key" :class="col.className">
            <component v-if="col.component" :is="col.component" :item="item" :col="col" />
            <span v-else>{{ getCellValue(item, col.key) }}</span>
          </td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(item, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">{{ emptyMessage }}</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div v-for="(menuItem, index) in contextMenuItems" :key="index">
        <!-- Single level menu items -->
        <div v-if="!menuItem.submenu" class="ctx-menu-item" :class="{ 'ctx-menu-danger': menuItem.danger }" @click="handleContextAction(menuItem)">
          <span class="ctx-menu-arrow"></span>{{ menuItem.label }}
        </div>
        <!-- Multi-level menu items -->
        <div v-else class="ctx-menu-item-group" @mouseenter="menuItem.show = true" @mouseleave="menuItem.show = false">
          <div class="ctx-menu-item ctx-menu-with-arrow">
            <span class="ctx-menu-arrow">‹</span> {{ menuItem.label }}
          </div>
          <div v-if="menuItem.show && menuItem.submenu" class="ctx-menu-sub">
            <div v-for="subItem in menuItem.submenu" :key="subItem.key" class="ctx-menu-sub-item" @click.stop="handleContextAction(subItem)">
              {{ subItem.label }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import ContentHeader from '../ContentHeader.vue'
import { ref } from 'vue'

export interface Column {
  key: string
  label: string
  className?: string
  component?: any
}

export interface ContextMenuItem {
  label: string
  action?: string
  danger?: boolean
  show?: boolean
  submenu?: ContextMenuItem[]
  handler?: (item: any) => void
  key?: string
}

const props = defineProps<{
  items: any[]
  search: string
  resourceTitle: string
  emptyMessage: string
  columns: Column[]
  contextMenuItems: ContextMenuItem[]
  getItemKey?: (item: any) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: any): void
  (e: 'contextAction', action: string, item: any, subAction?: string): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedItem = ref<any>(null)

function getCellValue(item: any, key: string): string {
  const keys = key.split('.')
  let value: any = item
  for (const k of keys) {
    value = value?.[k]
  }
  return value ?? '—'
}

function getItemKey(item: any): string {
  if (props.getItemKey) return props.getItemKey(item)
  const ns = (item as any).namespace
  return ns ? `${ns}${item.name}` : item.name
}

function openContextMenu(item: any, event: MouseEvent) {
  selectedItem.value = item
  
  // Reset all submenu flags
  props.contextMenuItems.forEach(menuItem => {
    menuItem.show = false
  })

  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = Math.min(300, props.contextMenuItems.length * 30 + 50)

  let left
  const availableLeft = rect.left
  if (availableLeft >= menuWidth) {
    left = rect.left - menuWidth + window.scrollX
  } else {
    left = rect.right + window.scrollX
    if (left + menuWidth > window.innerWidth + window.scrollX) {
      left = window.innerWidth - menuWidth + window.scrollX
    }
  }

  let top
  const availableBottom = window.innerHeight - rect.bottom
  if (availableBottom >= menuHeight) {
    top = rect.bottom + window.scrollY
  } else {
    top = rect.top - menuHeight + window.scrollY
    if (top < window.scrollY) {
      top = window.scrollY
    }
  }

  menuStyle.value = {
    position: 'absolute',
    top: `${top}px`,
    left: `${left}px`
  }
  showContextMenu.value = true
}

function hideContextMenu() {
  showContextMenu.value = false
}

function handleContextAction(menuItem: ContextMenuItem) {
  if (selectedItem.value) {
    if (menuItem.handler) {
      menuItem.handler(selectedItem.value)
    } else if (menuItem.action) {
      emit('contextAction', menuItem.action, selectedItem.value)
    }
  }
  hideContextMenu()
}

function handleRowClick(item: any) {
  emit('select', item)
}

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>

<style scoped>
.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 999;
  cursor: default;
}

.cluster-context-menu {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 1000;
  min-width: 160px;
}

.ctx-menu-item {
  padding: 6px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  padding-left: 34px;
  text-align: left;
  position: relative;
  margin: -9px;
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-item:first-child {
  border-radius: 3px 3px 0 0;
}

.ctx-menu-item:last-child {
  border-radius: 0 0 3px 3px;
}

.ctx-menu-arrow {
  position: absolute;
  left: 19px;
  color: #868d8a;
  font-size: 0.9rem;
}

.ctx-menu-item-group {
  position: relative;
  display: block;
}

.ctx-menu-with-arrow {
  padding-right: 20px;
}

.ctx-menu-sub {
  position: absolute;
  top: 0;
  right: 100%;
  margin-right: 8px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 1001;
  white-space: nowrap;
}

.ctx-menu-sub-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  white-space: nowrap;
}

.ctx-menu-sub-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-sub-item:first-child {
  border-radius: 3px 3px 0 0;
}

.ctx-menu-sub-item:last-child {
  border-radius: 0 0 3px 3px;
}

.ctx-menu-danger {
  color: #ef4444;
}

.ctx-menu-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

.clickable-row {
  cursor: pointer;
}

.clickable-row:hover {
  background: rgba(255, 255, 255, 0.02);
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

.empty {
  padding: 20px;
  text-align: center;
  color: #868d8a;
}
</style>
