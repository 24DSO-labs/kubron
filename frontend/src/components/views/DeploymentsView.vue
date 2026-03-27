<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <ContentHeader icon="" title="Deployments" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Ready</th><th>Up-to-date</th><th>Strategy</th><th>Images</th><th>Age</th><th></th></tr></thead>
      <tbody>
        <tr v-for="dep in items" :key="dep.namespace+dep.name" @click="handleShowDetails(dep)" class="clickable-row">
          <td class="resource-name">{{dep.name}}</td>
          <td class="ns-label">{{dep.namespace}}</td>
          <td><span :class="dep.readyReplicas===dep.replicas?'text-green':'text-yellow'">{{dep.readyReplicas}}/{{dep.replicas}}</span></td>
          <td>{{dep.updatedReplicas}}</td>
          <td>{{dep.strategy}}</td>
          <td class="images-cell"><div v-for="img in dep.images" :key="img" class="image-tag">{{img}}</div></td>
          <td>{{dep.age}}</td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(dep, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Deployments</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails(selectedDeployment)">
        <span class="ctx-menu-arrow"></span>Show Details
      </div>
      <div class="ctx-menu-item" @click="handleScale">
        <span class="ctx-menu-arrow"></span>Scale
      </div>
      <div class="ctx-menu-item" @click="handleRestart">
        <span class="ctx-menu-arrow"></span>Restart
      </div>
      <div class="ctx-menu-item" @click="handleLogs">
        <span class="ctx-menu-arrow"></span>Logs
      </div>
      <div class="ctx-menu-item" @click="handleEdit">
        <span class="ctx-menu-arrow"></span>Edit
      </div>
      <div class="ctx-menu-item ctx-menu-danger" @click="handleDelete">
        <span class="ctx-menu-arrow"></span>Delete
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { DeploymentInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
import { ref } from 'vue'

defineProps<{ items: DeploymentInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', dep: DeploymentInfo): void
  (e: 'scale', dep: DeploymentInfo, delta: number): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'edit', ns: string, name: string): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'delete', ns: string, name: string): void
}>()

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedDeployment = ref<DeploymentInfo | null>(null)

function openContextMenu(dep: DeploymentInfo, event: MouseEvent) {
  selectedDeployment.value = dep

  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = 250

  // Horizontal: prefer left (align right to button)
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

  // Vertical: prefer bottom
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

function handleShowDetails(dep: DeploymentInfo | null) {
  if (dep) emit('select', dep)
  hideContextMenu()
}

function handleScale() {
  if (selectedDeployment.value) {
    emit('scale', selectedDeployment.value, 1)
  }
  hideContextMenu()
}

function handleRestart() {
  if (selectedDeployment.value) {
    emit('restart', selectedDeployment.value.namespace, selectedDeployment.value.name)
  }
  hideContextMenu()
}

function handleLogs() {
  if (selectedDeployment.value) {
    emit('logs', selectedDeployment.value.namespace, selectedDeployment.value.name)
  }
  hideContextMenu()
}

function handleEdit() {
  if (selectedDeployment.value) {
    emit('edit', selectedDeployment.value.namespace, selectedDeployment.value.name)
  }
  hideContextMenu()
}

function handleDelete() {
  if (selectedDeployment.value) {
    emit('delete', selectedDeployment.value.namespace, selectedDeployment.value.name)
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
</style>
