<template>
  <div class="yaml-editor-panel">
    <div class="editor-header">
      <div class="editor-title">
        <span class="editor-icon">✏️</span>
        <span class="editor-resource">{{ resourceType }} <strong>{{ resourceName }}</strong></span>
        <span class="editor-namespace">in namespace <a href="#" class="ns-link">{{ namespace }}</a></span>
      </div>
      <div class="editor-actions">
        <button class="btn btn-sm btn-save" @click="$emit('apply')" title="Apply changes">
          💾 Save
        </button>
        <button class="btn btn-sm btn-secondary" @click="$emit('cancel')" title="Discard changes">
          ✕ Cancel
        </button>
        <button class="btn-close" @click="$emit('close')" title="Close editor">✕</button>
      </div>
    </div>

    <div ref="editorContainer" class="editor-container"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount } from 'vue'
import * as monaco from 'monaco-editor'
import * as yaml from 'js-yaml'

// Load Monaco Editor styles
import 'monaco-editor/esm/vs/editor/editor.main'
import 'monaco-editor/esm/vs/basic-languages/yaml/yaml'

// Import Codicon font
import codiconFont from 'monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf?url'

const props = defineProps<{
  content: string
  resourceType: string
  resourceName: string
  namespace: string
}>()

const emit = defineEmits<{
  (e: 'update:content', val: string): void
  (e: 'apply'): void
  (e: 'cancel'): void
  (e: 'close'): void
}>()

const editorContainer = ref<HTMLDivElement>()
let editor: monaco.editor.IStandaloneCodeEditor | null = null

onMounted(() => {
  // Set CSS variable for Codicon font
  const yamlPanel = document.querySelector('.yaml-editor-panel') as HTMLElement
  if (yamlPanel) {
    yamlPanel.style.setProperty('--codicon-font-url', `url('${codiconFont}')`)
  }

  if (editorContainer.value) {
    let yamlContent = props.content

    editor = monaco.editor.create(editorContainer.value, {
      value: yamlContent,
      language: 'yaml',
      theme: 'vs-dark',
      automaticLayout: true,
      minimap: {
        enabled: true,
        side: 'right',
      },
      scrollBeyondLastLine: false,
      wordWrap: 'off',
      fontSize: 12,
      fontFamily: "'Monaco', 'Menlo', 'Ubuntu Mono', monospace",
      lineNumbers: 'on',
      lineNumbersMinChars: 3,
      folding: true,
      formatOnPaste: false,
      formatOnType: false,
      glyphMargin: false,
      lineDecorationsWidth: 10,
      renderWhitespace: 'none',
      padding: {
        top: 0,
        bottom: 0,
      },
      'bracketPairColorization.enabled': false,
      'semanticHighlighting.enabled': false,
      'editor.defaultColorDecorators': false,
      tabSize: 2,
      insertSpaces: true,
      readOnly: false,
      unicodeHighlight: {
        ambiguousCharacters: false,
        invisibleCharacters: false,
        nonBasicASCII: false,
      },
      'editor.renderLineHighlight': 'none',
      'editor.occurrencesHighlight': 'off',
      'editor.selectionHighlight': false,
    } as any)

    // Disable all diagnostics and markers
    const model = editor.getModel()
    if (model) {
      // Clear all markers from all sources
      monaco.editor.setModelMarkers(model, 'owner', [])
      monaco.editor.setModelMarkers(model, 'typescript', [])
      monaco.editor.setModelMarkers(model, 'yaml', [])
      monaco.editor.setModelMarkers(model, 'jsonc', [])
    }

    // Set up listener to clear markers on every change
    editor.onDidChangeModelContent(() => {
      const currentModel = editor?.getModel()
      if (currentModel) {
        monaco.editor.setModelMarkers(currentModel, 'owner', [])
        monaco.editor.setModelMarkers(currentModel, 'typescript', [])
        monaco.editor.setModelMarkers(currentModel, 'yaml', [])
      }
    })

    editor.onDidChangeModelContent(() => {
      const newValue = editor?.getValue() || ''
      emit('update:content', newValue)
    })
  }
})

watch(
  () => props.content,
  (newContent) => {
    if (editor) {
      const currentValue = editor.getValue()
      if (currentValue !== newContent) {
        const model = editor.getModel()
        if (model) {
          model.setValue(newContent)
        } else {
          editor.setValue(newContent)
        }
      }
    }
  }
)

onBeforeUnmount(() => {
  editor?.dispose()
})
</script>

<style scoped>
.yaml-editor-panel {
  position: fixed;
  bottom: 0;
  right: 0;
  width: 100%;
  height: 60%;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  color: #d4d4d4;
  border-top: 1px solid #444;
  z-index: 100;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.5);
}

.editor-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 10px 16px;
  border-bottom: 1px solid #333;
  background: #252526;
  flex-shrink: 0;
  gap: 16px;
}

.editor-title {
  display: flex;
  align-items: center;
  gap: 8px;
  font-size: 14px;
  flex: 1;
  min-width: 0;
}

.editor-icon {
  font-size: 16px;
  flex-shrink: 0;
}

.editor-resource {
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-weight: 500;
}

.editor-resource strong {
  color: #60a5fa;
}

.editor-namespace {
  color: #6b6d8a;
  font-size: 12px;
  white-space: nowrap;
  flex-shrink: 0;
}

.ns-link {
  color: #60a5fa;
  text-decoration: none;
  cursor: pointer;
}

.ns-link:hover {
  text-decoration: underline;
}

.editor-actions {
  display: flex;
  gap: 8px;
  flex-shrink: 0;
}

.btn {
  padding: 4px 12px;
  font-size: 12px;
  background: #0e639c;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  transition: background 0.15s;
}

.btn:hover {
  background: #1177bb;
}

.btn-save {
  background: #007acc;
}

.btn-save:hover {
  background: #1084d7;
}

.btn-secondary {
  background: #444;
  color: #d4d4d4;
}

.btn-secondary:hover {
  background: #555;
}

.btn-close {
  padding: 4px 8px;
  font-size: 12px;
  background: transparent;
  color: #d4d4d4;
  border: 1px solid #444;
  border-radius: 3px;
  cursor: pointer;
  transition: all 0.15s;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #555;
}

.editor-container {
  flex: 1;
  overflow: hidden;
  margin: 0 !important;
  padding: 0 !important;
  width: 100%;
}

/* Monaco Editor integrations */
:deep(.monaco-editor) {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace !important;
}

:deep(.monaco-editor-background) {
  background: #1e1e1e;
}

:deep(.monaco-editor .margin) {
  background: #1a1a1a !important;
}

:deep(.monaco-editor .view-lines) {
  padding-left: 0 !important;
  margin-left: 0 !important;
  left: 0 !important;
}

:deep(.monaco-editor .view-line) {
  text-align: left !important;
}

:deep(.editor-widget) {
  background: #1e1e1e;
  border-color: #333;
}

/* Hide all error, warning, and info decorations */
:deep(.mtk31),
:deep(.squiggly-error),
:deep(.squiggly-warning),
:deep(.squiggly-info),
:deep(.error),
:deep(.warning),
:deep(.info) {
  border: none !important;
  text-decoration: none !important;
  background: none !important;
}

/* Remove wavy lines and red underlines */
:deep(.view-line span::before),
:deep(.view-line span[style*="border-bottom"]) {
  border: none !important;
  display: none !important;
}

/* Hide all decorations */
:deep([style*="border-bottom-style: wavy"]),
:deep([style*="border-bottom: 1px"]),
:deep([style*="red"]) {
  border: none !important;
  text-decoration: none !important;
}

/* Remove Monaco error decorations */
:deep(.cdr.error),
:deep(.cdr.warning),
:deep(.cdr.info),
:deep(.cdp.error),
:deep(.cdp.warning),
:deep(.cdp.info) {
  display: none !important;
  visibility: hidden !important;
}

/* Hide line decorations container if any */
:deep(.lines-decorations) {
  display: none !important;
}

/* Load Codicon font for Monaco icons */
@font-face {
  font-family: 'codicon';
  src: var(--codicon-font-url) format('truetype');
  font-display: swap;
}

:deep(.codicon) {
  font-family: 'codicon' !important;
  font-size: 16px;
  font-style: normal;
  font-weight: normal;
  font-variant: normal;
  text-transform: none;
  line-height: 1;
  display: inline-block;
  -webkit-font-smoothing: antialiased;
  -moz-osx-font-smoothing: grayscale;
}
</style>
