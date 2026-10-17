<template>
  <div>
    <div v-if="menu" class="menu-overlay" @click="hideMenu"></div>
    <ContentHeader icon="" :title="title" :count="items.length" :search="search" showSearch @refresh="emit('refresh')" @search="v => emit('search', v)" />

    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th v-for="col in columns" :key="col.key">{{ col.label }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="itemKey(item)" class="clickable-row" @click="emit('select', item)">
          <td v-for="col in columns" :key="col.key" :class="col.className">
            <component :is="col.component" v-if="col.component" :item="item" :col="col" />
            <span v-else :style="cellStyle(col, item)">{{ cellValue(col, item) }}</span>
          </td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openMenu(item, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">{{ emptyMessage }}</div>

    <!-- Контекстное меню целиком строится из menuFor(item): подменю контейнеров задают сами пункты -->
    <div v-if="menu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div
        v-for="(entry, i) in menu.entries"
        :key="entry.action"
        :class="['ctx-menu-item', { 'ctx-menu-danger': entry.danger }]"
        @mouseenter="onEnter(i)"
        @mouseleave="onLeave"
        @click="entry.submenu ? undefined : choose(entry.action)"
      >
        <span class="ctx-menu-arrow">{{ entry.submenu ? '‹' : '' }}</span>{{ entry.label }}
        <div v-if="entry.submenu && openSub === i" class="ctx-menu-sub">
          <div
            v-for="s in entry.submenu"
            :key="s.arg"
            class="ctx-menu-sub-item"
            @click.stop="choose(entry.action, s.arg)"
          >
            {{ s.label }}
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
/**
 * Таблица ресурсов с контекстным меню. Ничего не знает о конкретных действиях:
 * пункты и подменю приходят из menuFor(item), выбранный пункт уходит событием action.
 */
import { ref, onUnmounted } from 'vue'
import ContentHeader from '../ContentHeader.vue'
import type { ColumnDef, MenuItemDef } from '../../resources/types'

const props = defineProps<{
  items: any[]
  search: string
  title: string
  emptyMessage: string
  columns: ColumnDef[]
  menuFor: (item: any) => MenuItemDef[]
  getItemKey?: (item: any) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: any): void
  /** arg — выбранный элемент подменю (имя контейнера) */
  (e: 'action', action: string, item: any, arg?: string): void
}>()

// ─── Ячейки ───

function getCellValue(item: any, key: string): string {
  let value: any = item
  for (const k of key.split('.')) value = value?.[k]
  return value ?? '—'
}
const cellValue = (col: ColumnDef, item: any) => (col.formatter ? col.formatter(item) : getCellValue(item, col.key))
const cellStyle = (col: ColumnDef, item: any) => (col.colorFn ? { color: col.colorFn(item) } : {})
const itemKey = (item: any) => (props.getItemKey ? props.getItemKey(item) : `${item.namespace ?? ''}/${item.name}`)

// ─── Контекстное меню ───

const menu = ref<{ item: any; entries: MenuItemDef[] } | null>(null)
const menuStyle = ref<Record<string, string>>({})
const openSub = ref<number | null>(null)
let subTimer: ReturnType<typeof setTimeout> | null = null

function clearSubTimer() {
  if (subTimer) clearTimeout(subTimer)
  subTimer = null
}
onUnmounted(clearSubTimer)

function onEnter(i: number) {
  clearSubTimer()
  openSub.value = i
}
function onLeave() {
  clearSubTimer()
  subTimer = setTimeout(() => { openSub.value = null }, 150)
}

function openMenu(item: any, event: MouseEvent) {
  const entries = props.menuFor(item)
  openSub.value = null

  const rect = (event.target as HTMLElement).getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = Math.min(300, entries.length * 30 + 50)

  let left: number
  if (rect.left >= menuWidth) {
    left = rect.left - menuWidth + window.scrollX
  } else {
    left = rect.right + window.scrollX
    if (left + menuWidth > window.innerWidth + window.scrollX) {
      left = window.innerWidth - menuWidth + window.scrollX
    }
  }

  let top: number
  if (window.innerHeight - rect.bottom >= menuHeight) {
    top = rect.bottom + window.scrollY
  } else {
    top = Math.max(rect.top - menuHeight + window.scrollY, window.scrollY)
  }

  menuStyle.value = { position: 'absolute', top: `${top}px`, left: `${left}px` }
  menu.value = { item, entries }
}

function hideMenu() {
  clearSubTimer()
  menu.value = null
}

function choose(action: string, arg?: string) {
  const item = menu.value?.item
  hideMenu()
  if (item) emit('action', action, item, arg)
}
</script>

<style scoped>
.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 9998;
  cursor: default;
}

.cluster-context-menu {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 9999;
  min-width: 160px;
  overflow: visible;
  padding: 0;
  position: relative;
}

.ctx-menu-item {
  padding: 6px 12px 6px 34px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  text-align: left;
  position: relative;
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-item:first-child { border-radius: 3px 3px 0 0; }
.ctx-menu-item:last-child { border-radius: 0 0 3px 3px; }

.ctx-menu-arrow {
  position: absolute;
  left: 19px;
  color: #868d8a;
  font-size: 0.9rem;
}

.ctx-menu-sub {
  position: absolute;
  top: 0;
  right: 100%;
  margin-right: 4px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 10000;
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

.ctx-menu-sub-item:first-child { border-radius: 3px 3px 0 0; }
.ctx-menu-sub-item:last-child { border-radius: 0 0 3px 3px; }

.ctx-menu-danger { color: #ef4444; }
.ctx-menu-danger:hover { background: rgba(239, 68, 68, 0.2); }

.clickable-row { cursor: pointer; }
.clickable-row:hover { background: rgba(255, 255, 255, 0.02); }

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

td { text-align: left; }
</style>
