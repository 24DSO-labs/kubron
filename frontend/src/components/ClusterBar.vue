<template>
  <nav class="cluster-bar" aria-label="Кластеры">
    <div class="cluster-icons">
      <div
        v-for="cluster in clusters"
        :key="cluster.id"
        class="cluster-icon-wrap"
        :class="{ active: cluster.active, connected: cluster.connected }"
        role="button"
        tabindex="0"
        :title="tooltip(cluster)"
        :aria-label="cluster.displayName"
        :aria-current="cluster.active ? 'true' : undefined"
        @click="$emit('select', cluster.id)"
        @keydown.enter.prevent="$emit('select', cluster.id)"
        @keydown.space.prevent="$emit('select', cluster.id)"
        @contextmenu.prevent="onRightClick($event, cluster)"
      >
        <span class="cluster-pill"></span>
        <ClusterAvatar :name="cluster.displayName" :icon="cluster.icon" :color="cluster.color"
          :environment="cluster.environment" :size="40" show-env />
        <span class="cluster-dot" :class="cluster.connected ? 'dot-green' : 'dot-gray'"></span>
      </div>
    </div>

    <div class="cluster-bar-bottom">
      <div v-if="can(Perm.ClusterManage)" class="cluster-icon-wrap add-cluster" role="button" tabindex="0"
        title="Добавить кластер" aria-label="Добавить кластер" @click="$emit('add')"
        @keydown.enter.prevent="$emit('add')" @keydown.space.prevent="$emit('add')">
        <div class="cluster-icon cluster-add">+</div>
      </div>
    </div>

    <!-- Контекстное меню -->
    <div v-if="contextMenu" class="cluster-context-menu" role="menu"
      :style="{ top: contextMenu.y + 'px', left: contextMenu.x + 'px' }" @click.stop @contextmenu.prevent>
      <div class="ctx-menu-header">{{ contextMenu.cluster.displayName }}</div>
      <div class="ctx-menu-item" role="menuitem" @click="onEdit">Редактировать…</div>
      <div class="ctx-separator"></div>
      <div class="ctx-menu-item ctx-menu-danger" role="menuitem" @click="onDelete">Удалить…</div>
    </div>
  </nav>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, ref } from 'vue'
import type { ClusterStatus } from '../types'
import { can } from '../composables/useAuth'
import { Perm } from '../types/auth'
import { findEnvironment } from './cluster/clusterStyle'
import ClusterAvatar from './cluster/ClusterAvatar.vue'

defineProps<{ clusters: ClusterStatus[] }>()

const emit = defineEmits<{
  (e: 'select', id: string): void
  (e: 'add'): void
  (e: 'edit', id: string): void
  (e: 'delete', id: string): void
}>()

const MENU_WIDTH = 190
const MENU_HEIGHT = 110

const contextMenu = ref<{ x: number; y: number; cluster: ClusterStatus } | null>(null)

function tooltip(cluster: ClusterStatus): string {
  const env = findEnvironment(cluster.environment)
  return env ? `${cluster.displayName} · ${env.label}` : cluster.displayName
}

function onRightClick(e: MouseEvent, cluster: ClusterStatus) {
  if (!can(Perm.ClusterManage)) return
  // Меню появляется у курсора, но не уходит за край окна
  contextMenu.value = {
    x: Math.max(8, Math.min(e.clientX, window.innerWidth - MENU_WIDTH - 8)),
    y: Math.max(8, Math.min(e.clientY, window.innerHeight - MENU_HEIGHT - 8)),
    cluster,
  }
}

function closeMenu() {
  contextMenu.value = null
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') closeMenu()
}

function onEdit() {
  if (!contextMenu.value) return
  emit('edit', contextMenu.value.cluster.id)
  closeMenu()
}

function onDelete() {
  if (!contextMenu.value) return
  emit('delete', contextMenu.value.cluster.id)
  closeMenu()
}

onMounted(() => {
  window.addEventListener('click', closeMenu)
  window.addEventListener('keydown', onKeydown)
  window.addEventListener('blur', closeMenu)
  window.addEventListener('resize', closeMenu)
})

onBeforeUnmount(() => {
  window.removeEventListener('click', closeMenu)
  window.removeEventListener('keydown', onKeydown)
  window.removeEventListener('blur', closeMenu)
  window.removeEventListener('resize', closeMenu)
})
</script>
