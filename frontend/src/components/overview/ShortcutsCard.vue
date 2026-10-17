<template>
  <section class="ov-card shortcuts">
    <h2 class="ov-card-title">Shortcuts</h2>
    <ul class="list">
      <li v-for="s in SHORTCUTS" :key="s.view">
        <button type="button" @click="emit('navigate', s.view)">
          <span>{{ s.label }}</span>
          <kbd>Alt+{{ s.key.toUpperCase() }}</kbd>
        </button>
      </li>
    </ul>
  </section>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted } from 'vue'

const emit = defineEmits<{ (e: 'navigate', view: string): void }>()

/** view — те же идентификаторы, что в боковом меню (resources/navigation.ts) */
const SHORTCUTS = [
  { label: 'View all Pods', view: 'pods', key: 'p' },
  { label: 'View Deployments', view: 'deployments', key: 'd' },
  { label: 'View Nodes', view: 'nodes', key: 'n' },
  { label: 'View Events', view: 'events', key: 'e' },
  { label: 'Cluster settings', view: 'settings', key: 's' },
]

function isTyping(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  if (!el) return false
  return el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName)
}

function onKeydown(e: KeyboardEvent) {
  if (!e.altKey || e.ctrlKey || e.metaKey || e.shiftKey || isTyping(e.target)) return
  const hit = SHORTCUTS.find(s => s.key === e.key.toLowerCase())
  if (!hit) return
  e.preventDefault()
  emit('navigate', hit.view)
}

onMounted(() => window.addEventListener('keydown', onKeydown))
onUnmounted(() => window.removeEventListener('keydown', onKeydown))
</script>

<style scoped>
.list { list-style: none; display: flex; flex-direction: column; gap: 2px; }
.list button {
  width: 100%;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 9px 10px;
  background: none;
  border: none;
  border-radius: 8px;
  font: inherit;
  font-size: 13px;
  color: var(--ov-text);
  text-align: left;
  cursor: pointer;
}
.list button:hover, .list button:focus-visible { background: var(--ov-card-hover); outline: none; }
kbd {
  padding: 2px 6px;
  border: 1px solid var(--ov-border);
  border-radius: 5px;
  font: inherit;
  font-size: 11px;
  color: var(--ov-muted);
}
</style>
