<template>
  <div @click="hideContextMenu">
    <ContentHeader icon="" title="Namespaces" :count="filteredItems.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="filteredItems.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th><th>Status</th><th>Age</th><th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="namespace in filteredItems" :key="namespace.name" @click="$emit('select', namespace)" class="clickable-row">
          <td class="resource-name">{{ namespace.name }}</td>
          <td><span :class="textClass(namespace.status)">{{ namespace.status }}</span></td>
          <td>{{ namespace.age }}</td>
          <td>
            <button class="btn-icon" @click.stop="openContextMenu(namespace, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет namespaces</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <div class="ctx-menu-item" @click="handleShowDetails">Show Details</div>
      <div class="ctx-menu-item ctx-menu-danger" @click="handleDelete">Delete</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { NamespaceInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
import { ref, computed, onMounted } from 'vue'
import { useK8s } from '../../composables/useK8s'

const props = defineProps<{
  items?: NamespaceInfo[]
  search?: string
  statusClass: (s: string) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', namespace: NamespaceInfo): void
  (e: 'delete', namespace: NamespaceInfo): void
}>()

const k8s = useK8s()
const localSearch = ref('')

const items = computed(() => props.items || k8s.namespaces.value || [])
const search = computed(() => props.search || localSearch.value)

const filteredItems = computed(() => {
  const s = search.value.toLowerCase()
  if (!s) return items.value
  return items.value.filter(ns => ns.name.toLowerCase().includes(s))
})

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedNamespace = ref<NamespaceInfo | null>(null)

onMounted(async () => {
  if (!props.items) {
    await k8s.loadNamespaces()
  }
})

function openContextMenu(namespace: NamespaceInfo, event: MouseEvent) {
  selectedNamespace.value = namespace
  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = 100

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

function handleShowDetails() {
  if (selectedNamespace.value) emit('select', selectedNamespace.value)
  hideContextMenu()
}

function handleDelete() {
  if (selectedNamespace.value) emit('delete', selectedNamespace.value)
  hideContextMenu()
}

function textClass(s: string): string {
  return `text-${s.toLowerCase()}`
}

function onRefresh() { emit('refresh') }

function onSearch(val: string) {
  localSearch.value = val
  emit('search', val)
}
</script>
