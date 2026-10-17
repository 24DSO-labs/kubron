<template>
  <div>
    <div v-if="menu" class="menu-overlay" @click="hideMenu" @contextmenu.prevent="hideMenu"></div>
    <ContentHeader :icon="icon" :title="title" :count="items.length" :search="search" showSearch @refresh="emit('refresh')" @search="v => emit('search', v)" />

    <table
  v-if="items.length > 0"
  ref="tableRef"
  class="table"
  :class="{ 'table-fixed': fixed }"
  :style="fixed ? { width: totalWidth + 'px' } : undefined"
>
  <colgroup v-if="fixed">
  <col v-for="col in columns" :key="col.key" :style="{ width: widths[col.key] + 'px' }" />
  <col :style="{ width: ACTION_COL + 'px' }" />
</colgroup>
<thead>
  <tr>
    <th v-for="col in columns" :key="col.key" :data-col="col.key">
      {{ col.label }}
      <span
        class="col-resizer"
        :class="{ active: resizingKey === col.key }"
        title="Двойной клик — сбросить ширину"
        @pointerdown.prevent.stop="startResize(col.key, $event)"
        @pointermove="onDrag"
        @pointerup="stopDrag"
        @pointercancel="stopDrag"
        @dblclick.stop="resetWidths"
      ></span>
    </th>
    <th></th>
  </tr>
</thead>
      <tbody>
        <tr v-for="item in items" :key="itemKey(item)" class="clickable-row" @click="emit('select', item)" @contextmenu.prevent="openMenuAt(item, $event)">
          <td v-for="col in columns" :key="col.key" :class="col.className">
            <component :is="col.component" v-if="col.component" :item="item" :col="col" />
<span v-else-if="col.animated" :style="cellStyle(col, item)">
  <RollingValue :value="String(cellValue(col, item))" />
</span>
            <span v-else-if="col.link" :style="cellStyle(col, item)">
              <template v-if="col.link(item)">
                <a
                  class="rd-link"
                  @click.stop="emit('open', col.link(item)!)"
                >
                  {{ col.linkLabel ? (col.linkLabel(item) ?? cellValue(col, item)) : cellValue(col, item) }}
                </a>
              </template>
              <template v-else>{{ cellValue(col, item) }}</template>
            </span>
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

import ContentHeader from '../ContentHeader.vue'
import type { ColumnDef, MenuItemDef } from '../../resources/types'
import './resource.css'
import { ref, reactive, computed, watch, nextTick, onMounted, onUnmounted } from 'vue'
import RollingValue from './RollingValue.vue'

const props = defineProps<{
  items: any[]
  search: string
  title: string
  icon?: string
  emptyMessage: string
  columns: ColumnDef[]
  menuFor: (item: any) => MenuItemDef[]
  getItemKey?: (item: any) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: any): void
  /** Переход к другому ресурсу по ссылке в ячейке */
  (e: 'open', data: { id: string; ns: string; name: string }): void
  /** arg — выбранный элемент подменю (имя контейнера) */
  (e: 'action', action: string, item: any, arg?: string): void
}>()




// ─── Ширина колонок ───
const MIN_COL = 50
const ACTION_COL = 56
const storageKey = () => `kubron:colwidths:${props.title}`

const tableRef = ref<HTMLTableElement | null>(null)
const widths = reactive<Record<string, number>>({})
const fixed = ref(false)
const resizingKey = ref<string | null>(null)

const totalWidth = computed(() =>
  props.columns.reduce((s, c) => s + (widths[c.key] ?? MIN_COL), ACTION_COL),
)

function loadSaved() {
  try {
    const saved = JSON.parse(localStorage.getItem(storageKey()) || 'null')
    if (saved && props.columns.every(c => typeof saved[c.key] === 'number')) {
      Object.assign(widths, saved)
      fixed.value = true
    }
  } catch { /* ignore */ }
}
function saveWidths() {
  try { localStorage.setItem(storageKey(), JSON.stringify(widths)) } catch { /* ignore */ }
}

/** Замеряем ширины в авто-раскладке и фиксируем; таблица не выходит за ширину контейнера */
async function freeze() {
  await nextTick()
  const table = tableRef.value
  const ths = table?.querySelectorAll<HTMLElement>('th[data-col]')
  if (!table || !ths?.length) return
  ths.forEach(th => {
    const key = th.dataset.col!
    const def = props.columns.find(c => c.key === key)
    widths[key] = def?.width ?? Math.max(MIN_COL, Math.ceil(th.getBoundingClientRect().width))
  })

  const avail = table.parentElement?.clientWidth ?? 0
  const excess = totalWidth.value - avail
  if (avail && excess > 0) {
    const widest = props.columns.reduce((a, c) => (widths[c.key] > widths[a.key] ? c : a))
    widths[widest.key] = Math.max(MIN_COL, widths[widest.key] - excess)
  }
  fixed.value = true
}

function resetWidths() {
  try { localStorage.removeItem(storageKey()) } catch { /* ignore */ }
  fixed.value = false
  freeze()
}

let drag: { key: string; startX: number; startW: number } | null = null

function startResize(key: string, e: PointerEvent) {
  if (!fixed.value) return
  ;(e.currentTarget as HTMLElement).setPointerCapture(e.pointerId)
  drag = { key, startX: e.clientX, startW: widths[key] }
  resizingKey.value = key
  document.body.style.cursor = 'col-resize'
  document.body.style.userSelect = 'none'
}
function onDrag(e: PointerEvent) {
  if (!drag) return
  widths[drag.key] = Math.max(MIN_COL, drag.startW + e.clientX - drag.startX)
}
function stopDrag() {
  if (!drag) return
  drag = null
  resizingKey.value = null
  document.body.style.cursor = ''
  document.body.style.userSelect = ''
  saveWidths()
}

loadSaved()
onMounted(() => { if (props.items.length && !fixed.value) freeze() })
watch(() => props.items.length, n => { if (n > 0 && !fixed.value) freeze() })

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
function onKey(e: KeyboardEvent) {
  if (e.key === 'Escape') hideMenu()
}
onMounted(() => window.addEventListener('keydown', onKey))
onUnmounted(() => {
  window.removeEventListener('keydown', onKey)
  clearSubTimer()
})

function onEnter(i: number) {
  clearSubTimer()
  openSub.value = i
}
function onLeave() {
  clearSubTimer()
  subTimer = setTimeout(() => { openSub.value = null }, 150)
}

const MENU_WIDTH = 200

function showMenu(item: any, x: number, y: number) {
  const entries = props.menuFor(item)
  openSub.value = null

  const menuHeight = Math.min(300, entries.length * 30 + 50)

  // не выходить за правый/нижний край окна
  const left = Math.max(0, Math.min(x, window.innerWidth - MENU_WIDTH))
  const top = Math.max(0, Math.min(y, window.innerHeight - menuHeight))

  menuStyle.value = { position: 'fixed', top: `${top}px`, left: `${left}px` }
  menu.value = { item, entries }
}

/** Правая кнопка мыши: меню у курсора */
function openMenuAt(item: any, event: MouseEvent) {
  showMenu(item, event.clientX, event.clientY)
}

/** Кнопка «⋮»: меню слева от кнопки (как раньше) */
function openMenu(item: any, event: MouseEvent) {
  const rect = (event.currentTarget as HTMLElement).getBoundingClientRect()
  const x = rect.left >= MENU_WIDTH ? rect.left - MENU_WIDTH : rect.right
  showMenu(item, x, rect.bottom)
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

.table-fixed { table-layout: fixed; }
.table-fixed th, .table-fixed td {
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
}
.col-resizer {
  position: absolute; top: 0; bottom: 0; right: 0; width: 8px;
  cursor: col-resize; touch-action: none;
}
.col-resizer::after {
  content: ''; position: absolute; top: 25%; bottom: 25%; right: 3px; width: 1px;
  background: #45475a;
}
.col-resizer:hover, .col-resizer.active { background: rgba(137, 180, 250, 0.35); }
.col-resizer:hover::after, .col-resizer.active::after { background: #89b4fa; }
</style>
