<template>
  <div class="cluster-bar">
    <div class="cluster-icons">
      <div
        v-for="cluster in clusters"
        :key="cluster.id"
        class="cluster-icon-wrap"
        :class="{ active: cluster.active, connected: cluster.connected }"
        :title="cluster.displayName"
        @click="$emit('select', cluster.id)"
        @contextmenu.prevent="onRightClick(cluster)"
      >
        <div class="cluster-icon" :style="{ borderColor: cluster.active ? cluster.color : 'transparent' }">
          {{ cluster.icon }}
        </div>
        <div class="cluster-dot" :class="cluster.connected ? 'dot-green' : 'dot-gray'"></div>
      </div>
    </div>

    <div class="cluster-bar-bottom">
      <div class="cluster-icon-wrap add-cluster" title="Добавить кластер" @click="$emit('add')">
        <div class="cluster-icon cluster-add">+</div>
      </div>
    </div>

    <!-- Context menu -->
    <div v-if="contextMenu" class="cluster-context-menu" :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }">
      <div class="ctx-menu-header">{{ contextMenu.cluster.displayName }}</div>
      <div class="ctx-menu-item" @click="onEdit">✏️ Редактировать</div>
      <div class="ctx-menu-item ctx-menu-danger" @click="onDelete">🗑 Удалить</div>
      <div class="ctx-menu-item" @click="contextMenu = null">✕ Закрыть</div>
    </div>

    <!-- Edit dialog -->
    <div v-if="editCluster" class="cluster-edit-overlay" @click.self="editCluster = null">
      <div class="cluster-edit-dialog">
        <h3>Редактировать кластер</h3>
        <div class="form-row">
          <label>Имя:</label>
          <input v-model="editName" class="form-input" />
        </div>
        <div class="form-row">
          <label>Иконка:</label>
          <div class="icon-picker">
            <span v-for="ic in icons" :key="ic" class="icon-option" :class="{ selected: editIcon === ic }" @click="editIcon = ic">{{ ic }}</span>
          </div>
        </div>
        <div class="form-row">
          <label>Цвет:</label>
          <div class="color-picker">
            <span v-for="cl in colors" :key="cl" class="color-option" :class="{ selected: editColor === cl }" :style="{ background: cl }" @click="editColor = cl"></span>
          </div>
        </div>
        <div class="form-actions">
          <button class="btn btn-sm" @click="saveEdit">💾 Сохранить</button>
          <button class="btn btn-sm btn-secondary" @click="editCluster = null">Отмена</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import type { ClusterStatus } from '../types'

defineProps<{ clusters: ClusterStatus[] }>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'add'): void
  (e: 'delete', id: string): void
  (e: 'update', id: string, name: string, icon: string, color: string): void
}>()

const icons = ['🟢', '🔵', '🟡', '🔴', '🟣', '⚪', '🏗', '🚀', '⭐', '🔥', '💎', '🌍']
const colors = ['#a6e3a1', '#89b4fa', '#f9e2af', '#f38ba8', '#cba6f7', '#94e2d5', '#fab387', '#74c7ec']

const contextMenu = ref<{ x: number; y: number; cluster: ClusterStatus } | null>(null)
const editCluster = ref<ClusterStatus | null>(null)
const editName = ref('')
const editIcon = ref('')
const editColor = ref('')

function onRightClick(cluster: ClusterStatus) {
  contextMenu.value = { x: 70, y: 100, cluster }
}

function onEdit() {
  if (!contextMenu.value) return
  editCluster.value = contextMenu.value.cluster
  editName.value = contextMenu.value.cluster.displayName
  editIcon.value = contextMenu.value.cluster.icon
  editColor.value = contextMenu.value.cluster.color
  contextMenu.value = null
}

function onDelete() {
  if (!contextMenu.value) return
  emit('delete', contextMenu.value.cluster.id)
  contextMenu.value = null
}

function saveEdit() {
  if (!editCluster.value) return
  emit('update', editCluster.value.id, editName.value, editIcon.value, editColor.value)
  editCluster.value = null
}
</script>