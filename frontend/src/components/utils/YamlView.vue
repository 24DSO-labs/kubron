<template>
<div class="yaml-view">
  <div class="editor-header">
    <div class="editor-title">
      <span v-if="isCreate">New <strong>{{ tab.kindLabel }}</strong></span>
      <span v-else>{{ tab.resourceType }} <strong>{{ tab.resourceName }}</strong></span>
      <span v-if="tab.namespace" class="editor-namespace">in <a href="#" class="ns-link">{{ tab.namespace
          }}</a></span>
      <span v-if="isDirty && !isCreate" class="editor-dirty" title="Есть несохранённые изменения">●</span>
      <span v-if="tab.view === 'diff'" class="editor-mode">Review changes</span>
    </div>
    <div class="editor-actions">
      <template v-if="isCreate">
        <span class="editor-hint">Измените манифест и создайте объект</span>
        <button class="btn btn-sm btn-save"
          :disabled="tab.saving || !hasContent"
          title="Создать объект в кластере (Ctrl+S)"
          @click="applyChanges()">{{ tab.saving ? '⏳ Creating...' : '✓ Create' }}</button>
      </template>
      <template v-else-if="tab.view === 'diff'">
        <button class="btn btn-sm btn-secondary" @click="backToEdit()">← Back to editing</button>
        <button class="btn btn-sm btn-save"
          :disabled="tab.saving || !isDirty"
          @click="applyChanges()">{{ tab.saving ? '⏳ Applying...' : '✓ Apply' }}</button>
      </template>
      <template v-else>
        <button class="btn btn-sm btn-secondary" :disabled="tab.loading || tab.saving"
          title="Перечитать манифест из кластера" @click="reloadYaml()">{{ tab.loading ? '⏳ Loading...' : '↻ Reload' }}</button>
        <span v-if="!canEdit" class="editor-readonly" title="Для изменения ресурсов нужны права администратора">Только чтение</span>
        <button v-else class="btn btn-sm btn-save"
          :disabled="tab.saving || !isDirty"
          title="Посмотреть изменения и применить (Ctrl+S)"
          @click="reviewChanges()">💾 Review &amp; Save</button>
      </template>
    </div>
  </div>
  <div v-show="tab.view !== 'diff'" ref="editorEl"
    class="editor-container"></div>
  <div v-if="tab.view === 'diff'" ref="diffEl"
    class="editor-container"></div>
</div>
</template>

<script lang="ts" setup>
import { ref, computed, watch, nextTick, onMounted, onBeforeUnmount } from 'vue'
import * as monaco from 'monaco-editor'
import { GetResourceYAML } from '../../../wailsjs/go/main/App'
import { useNotifications } from '../../composables/useNotifications'
import { can } from '../../composables/useAuth'
import { Perm } from '../../types/auth'
import { askConfirm } from '../../resources/confirm'
import type { YamlTab } from '../../composables/useTabbedPanel'

const props = defineProps<{ tab: YamlTab; active: boolean }>()
const emit = defineEmits<{ (e: 'apply', tabId: string): void }>()

const notifications = useNotifications()

// Без права на запись YAML открывается только для просмотра
const canEdit = computed(() => can(Perm.Write))
const isDirty = computed(() => props.tab.content !== props.tab.originalContent)
// Новый объект: сравнивать не с чем, сразу создаём (без просмотра diff)
const isCreate = computed(() => props.tab.mode === 'create')
const hasContent = computed(() => props.tab.content.trim() !== '')

const editorEl = ref<HTMLDivElement | null>(null)
const diffEl = ref<HTMLDivElement | null>(null)

let editor: monaco.editor.IStandaloneCodeEditor | null = null
let diff: {
  editor: monaco.editor.IStandaloneDiffEditor
  original: monaco.editor.ITextModel
  modified: monaco.editor.ITextModel
} | null = null

const FONT = "'Monaco', 'Menlo', monospace"

function createEditor() {
  if (!editorEl.value || editor) return
  editor = monaco.editor.create(editorEl.value, {
    value: props.tab.content,
    language: 'yaml',
    theme: 'vs-dark',
    automaticLayout: true,
    minimap: { enabled: true },
    scrollBeyondLastLine: false,
    wordWrap: 'off',
    fontSize: 12,
    fontFamily: FONT,
    lineNumbers: 'on',
    folding: true,
    readOnly: !canEdit.value,
  })
  editor.addCommand(monaco.KeyMod.CtrlCmd | monaco.KeyCode.KeyS, () => reviewChanges())
  editor.onDidChangeModelContent(() => { props.tab.content = editor!.getValue() })
}

/** Diff «в кластере сейчас» (слева) ↔ «ваша версия» (справа); только просмотр */
function createDiff() {
  if (!diffEl.value || diff) return
  const original = monaco.editor.createModel(props.tab.originalContent, 'yaml')
  const modified = monaco.editor.createModel(props.tab.content, 'yaml')
  const ed = monaco.editor.createDiffEditor(diffEl.value, {
    theme: 'vs-dark',
    automaticLayout: true,
    readOnly: true,
    originalEditable: false,
    renderSideBySide: true,
    ignoreTrimWhitespace: false,
    scrollBeyondLastLine: false,
    minimap: { enabled: false },
    fontSize: 12,
    fontFamily: FONT,
  })
  ed.setModel({ original, modified })
  diff = { editor: ed, original, modified }
}

function disposeDiff() {
  if (!diff) return
  diff.editor.dispose()
  diff.original.dispose()
  diff.modified.dispose()
  diff = null
}

onMounted(createEditor)

onBeforeUnmount(() => {
  disposeDiff()
  editor?.dispose()
  editor = null
})

// Режим просмотра изменений: diff-редактор живёт, пока открыт этот режим
watch(() => props.tab.view, async view => {
  await nextTick()
  if (view === 'diff') createDiff()
  else { disposeDiff(); editor?.layout() }
})

// Внешнее изменение content вкладки (не набор текста пользователем) -> обновить редактор
watch(() => props.tab.content, content => {
  if (editor && editor.getValue() !== content) editor.setValue(content)
})

watch(canEdit, v => editor?.updateOptions({ readOnly: !v }))

// Вкладка снова видна: редактору нужен пересчёт размеров
watch(() => props.active, async active => {
  if (!active) return
  await nextTick()
  editor?.layout()
})

// Изменения применены (или откачены): смотреть в diff больше нечего
watch(isDirty, dirty => {
  if (!dirty && props.tab.view === 'diff') props.tab.view = 'edit'
})

/** Сохранение начинается с просмотра изменений (Save / Ctrl+S) */
function reviewChanges() {
  if (isCreate.value) return applyChanges()
  if (!canEdit.value || props.tab.saving || !isDirty.value) return
  props.tab.view = 'diff'
}

/** Применить в кластере (кнопка Apply в режиме diff); результат обрабатывает App.vue */
function applyChanges() {
  if (!canEdit.value || props.tab.saving) return
  if (isCreate.value ? !hasContent.value : !isDirty.value) return
  emit('apply', props.tab.id)
}

function backToEdit() {
  props.tab.view = 'edit'
  nextTick(() => {
    editor?.layout()
    editor?.focus()
  })
}

/** Перечитать манифест из кластера; несохранённые правки — только после подтверждения */
async function reloadYaml() {
  const tab = props.tab
  if (tab.loading || tab.saving) return
  if (isDirty.value && !(await askConfirm('Discard unsaved changes and reload YAML from the cluster?'))) return
  tab.loading = true
  try {
    const fresh = await GetResourceYAML(tab.namespace, tab.resourceType, tab.resourceName, true)
    tab.originalContent = fresh
    tab.content = fresh
    tab.view = 'edit'
  } catch (e: any) {
    notifications.notifyError(`Не удалось перечитать YAML: ${e?.message || e}`)
  } finally {
    tab.loading = false
  }
}
</script>

<style scoped>
/* YAML View */
.editor-readonly {
  font-size: 12px;
  color: #f9e2af;
  background: rgba(249, 226, 175, 0.12);
  border-radius: 999px;
  padding: 3px 10px;
}

.yaml-view {
  display: flex;
  flex-direction: column;
  height: 100%;
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #252526;
}

.editor-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
}

.editor-actions {
  display: flex;
  align-items: center;
  gap: 8px;
}

.editor-mode {
  font-size: 12px;
  color: #89b4fa;
  background: rgba(137, 180, 250, 0.12);
  border-radius: 999px;
  padding: 2px 10px;
}

.editor-hint { font-size: 12px; color: #6b6d8a; }

.editor-namespace {
  color: #6b6d8a;
  font-size: 12px;
}

.ns-link {
  color: #60a5fa;
  text-decoration: none;
}

.ns-link:hover {
  text-decoration: underline;
}

.btn {
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
}

.btn-sm {
  padding: 4px 8px;
}

.btn-save {
  background: #10b981;
  color: white;
}

.btn-save:hover {
  background: #059669;
}

.editor-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}


:deep(.monaco-editor) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .margin) {
  background-color: #1a1b26 !important;
}

:deep(.monaco-editor .minimap) {
  border-left: 1px solid #3b3d5c;
}

.editor-dirty { color: #facc15; font-size: 14px; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>
