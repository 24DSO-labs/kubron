<template>
  <div ref="root" class="cm">
    <button
      class="cm-btn"
      type="button"
      aria-haspopup="true"
      :aria-expanded="open"
      @click="open = !open"
    >
      Колонки
    </button>
    <div v-if="open" class="cm-pop" role="dialog" aria-label="Настройка колонок">
      <ul class="cm-list">
        <li
          v-for="(c, i) in ordered"
          :key="c.key"
          class="cm-row"
          :class="{ over: overIdx === i }"
          draggable="true"
          @dragstart="onDragStart(i, $event)"
          @dragover.prevent="overIdx = i"
          @dragleave="overIdx = null"
          @drop.prevent="onDrop(i)"
          @dragend="dragFrom = overIdx = null"
        >
          <label class="cm-label">
            <input
              type="checkbox"
              :checked="!hidden.includes(c.key)"
              :disabled="c.key === required"
              @change="emit('toggle', c.key)"
            />
            <span>{{ c.label }}</span>
          </label>
          <span class="cm-move">
            <button type="button" :disabled="i === 0" aria-label="Выше" @click="emit('move', i, i - 1)">↑</button>
            <button type="button" :disabled="i === ordered.length - 1" aria-label="Ниже" @click="emit('move', i, i + 1)">↓</button>
          </span>
        </li>
      </ul>
      <button class="cm-reset" type="button" @click="emit('reset')">Сбросить колонки</button>
    </div>
  </div>
</template>

<script lang="ts" setup>
/** Выпадающее меню: видимость колонок (чекбокс) и порядок (перетаскивание или кнопки ↑/↓). Состояния не хранит. */
import { onMounted, onUnmounted, ref } from 'vue'

defineProps<{
  ordered: { key: string; label: string }[]
  hidden: string[]
  /** Ключ колонки, которую нельзя скрыть (имя) */
  required: string
}>()

const emit = defineEmits<{
  (e: 'toggle', key: string): void
  (e: 'move', from: number, to: number): void
  (e: 'reset'): void
}>()

const open = ref(false)
const root = ref<HTMLElement | null>(null)
const dragFrom = ref<number | null>(null)
const overIdx = ref<number | null>(null)

function onDragStart(i: number, e: DragEvent) {
  dragFrom.value = i
  if (e.dataTransfer) {
    e.dataTransfer.effectAllowed = 'move'
    e.dataTransfer.setData('text/plain', String(i)) // Firefox/WebKit не начинают drag без данных
  }
}

function onDrop(to: number) {
  if (dragFrom.value !== null && dragFrom.value !== to) emit('move', dragFrom.value, to)
  dragFrom.value = overIdx.value = null
}

const onDocClick = (e: MouseEvent) => {
  if (!root.value?.contains(e.target as Node)) open.value = false
}
const onKey = (e: KeyboardEvent) => {
  if (e.key === 'Escape') open.value = false
}
onMounted(() => {
  document.addEventListener('click', onDocClick)
  document.addEventListener('keydown', onKey)
})
onUnmounted(() => {
  document.removeEventListener('click', onDocClick)
  document.removeEventListener('keydown', onKey)
})
</script>

<style scoped>
.cm { position: relative; }
.cm-btn {
  padding: 7px 14px;
  background: #1a1d2e;
  border: 1px solid #3b3d5c;
  border-radius: 8px;
  font: inherit;
  font-size: 13px;
  font-weight: 500;
  color: #e6e8f5;
  cursor: pointer;
  transition: border-color 0.15s;
}
.cm-btn:hover { border-color: #89b4fa; }
.cm-pop {
  position: absolute;
  right: 0;
  top: calc(100% + 6px);
  z-index: 200;
  min-width: 240px;
  max-height: 60vh;
  overflow: auto;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 6px;
  padding: 6px;
  text-align: left;
  box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
}
.cm-list { list-style: none; margin: 0; padding: 0; }
.cm-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 4px 6px;
  border-radius: 4px;
  cursor: grab;
}
.cm-row:hover { background: rgba(255, 255, 255, 0.06); }
.cm-row.over { outline: 1px dashed #89b4fa; }
.cm-label { display: flex; align-items: center; gap: 8px; flex: 1; font-size: 13px; color: #d4d4d4; }
.cm-label input { accent-color: #89b4fa; margin: 0; }
.cm-move button {
  width: 22px;
  height: 22px;
  margin-left: 2px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: #d4d4d4;
  cursor: pointer;
}
.cm-move button:hover:not(:disabled) { border-color: #89b4fa; }
.cm-move button:disabled { opacity: 0.3; cursor: default; }
.cm-reset {
  width: 100%;
  margin-top: 6px;
  padding: 6px;
  background: transparent;
  border: 0;
  border-top: 1px solid #3b3d5c;
  color: #89b4fa;
  font: inherit;
  font-size: 12px;
  cursor: pointer;
}
</style>
