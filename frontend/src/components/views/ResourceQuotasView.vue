<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <ContentHeader icon="" title="ResourceQuotas" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Namespace</th><th>CPU Limit</th><th>CPU Used</th><th>Memory Limit</th><th>Memory Used</th><th>Age</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="rq in items" :key="rq.namespace+rq.name" @click="$emit('select', rq)" class="clickable-row">
          <td class="resource-name">{{ rq.name }}</td>
          <td class="ns-label">{{ rq.namespace }}</td>
          <td>{{ rq.cpuLimit }}</td>
          <td>{{ rq.cpuUsed }}</td>
          <td>{{ rq.memLimit }}</td>
          <td>{{ rq.memUsed }}</td>
          <td>{{ rq.age }}</td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(rq, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет ResourceQuotas</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails">
        <span class="ctx-menu-arrow"></span>Show Details
      </div>
      <div class="ctx-menu-item" @click="handleYAML">
        <span class="ctx-menu-arrow"></span>View YAML
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ResourceQuotaInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
import { ref } from 'vue'

defineProps<{ items: ResourceQuotaInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', rq: ResourceQuotaInfo): void
  (e: 'yaml', ns: string, name: string): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedRQ = ref<ResourceQuotaInfo | null>(null)

function openContextMenu(rq: ResourceQuotaInfo, event: MouseEvent) {
  selectedRQ.value = rq

  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = 120

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

function handleShowDetails() {
  if (selectedRQ.value) emit('select', selectedRQ.value)
  hideContextMenu()
}

function handleYAML() {
  if (selectedRQ.value) {
    emit('yaml', selectedRQ.value.namespace, selectedRQ.value.name)
  }
  hideContextMenu()
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
</style>
