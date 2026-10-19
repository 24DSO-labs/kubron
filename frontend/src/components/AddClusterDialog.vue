<template>
  <div class="dialog-overlay" @click.self="$emit('close')">
    <div class="dialog">
      <div class="dialog-header">
        <h2>➕ Добавить кластер</h2>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>

      <div class="dialog-body">
        <!-- Шаг 1: Имя и стиль -->
        <div class="form-row">
          <label>Название кластера:</label>
          <input v-model="displayName" placeholder="Production" class="form-input" />
        </div>

        <div class="form-row">
          <label>Иконка:</label>
          <div class="icon-picker">
            <span v-for="ic in icons" :key="ic" class="icon-option" :class="{ selected: icon === ic }" @click="icon = ic">{{ ic }}</span>
          </div>
        </div>

        <div class="form-row">
          <label>Цвет:</label>
          <div class="color-picker">
            <span v-for="cl in colors" :key="cl" class="color-option" :class="{ selected: color === cl }" :style="{ background: cl }" @click="color = cl"></span>
          </div>
        </div>

        <!-- Шаг 2: Способ добавления -->
        <div class="form-row">
          <label>Kubeconfig:</label>
          <div class="add-method-tabs">
            <button :class="['tab', { active: method === 'file' }]" @click="method = 'file'">📂 Файл</button>
            <button :class="['tab', { active: method === 'paste' }]" @click="method = 'paste'">📋 Вставить</button>
          </div>
        </div>

        <div v-if="method === 'file'" class="form-row">
          <button @click="selectFile" class="btn">📂 Выбрать kubeconfig файл</button>
          <span v-if="fileSelected" class="file-selected">✅ Файл выбран</span>
        </div>

        <div v-if="method === 'paste'" class="form-row">
          <textarea v-model="pasteText" placeholder="Вставьте содержимое kubeconfig..." class="form-textarea"></textarea>
        </div>

        <div v-if="error" class="form-error">❌ {{ error }}</div>
      </div>

      <div class="dialog-footer">
        <button class="btn" @click="doAdd" :disabled="!canAdd">➕ Добавить</button>
        <button class="btn btn-secondary" @click="$emit('close')">Отмена</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

const props = defineProps<{ pickFile: () => Promise<string> }>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'addFile', path: string, name: string, icon: string, color: string): void
  (e: 'addText', text: string, name: string, icon: string, color: string): void
}>()

const icons = ['🟢', '🔵', '🟡', '🔴', '🟣', '⚪', '🏗', '🚀', '⭐', '🔥', '💎', '🌍']
const colors = ['#a6e3a1', '#89b4fa', '#f9e2af', '#f38ba8', '#cba6f7', '#94e2d5', '#fab387', '#74c7ec']

const displayName = ref('')
const icon = ref('🔵')
const color = ref('#89b4fa')
const method = ref<'file' | 'paste'>('file')
const pasteText = ref('')
const filePath = ref('')
const fileSelected = ref(false)
const error = ref('')

const canAdd = computed(() => {
  if (!displayName.value.trim()) return false
  if (method.value === 'paste' && !pasteText.value.trim()) return false
  if (method.value === 'file' && !filePath.value) return false
  return true
})

async function selectFile() {
  const path = await props.pickFile()
  if (path) {
    filePath.value = path
    fileSelected.value = true
  }
}

function doAdd() {
  if (method.value === 'file') {
    emit('addFile', filePath.value, displayName.value, icon.value, color.value)
  } else {
    emit('addText', pasteText.value, displayName.value, icon.value, color.value)
  }
}
</script>