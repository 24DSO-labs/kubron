<template>
  <div class="tabbed-editor-wrapper" :style="{ bottom: 0, right: 0, width: '100%', height: '100%', position: 'fixed', zIndex: 50 }">
    <div class="tabbed-editor-panel">
      <div class="resize-handle resize-top" @mousedown="(e) => startResize(e, 'top')"></div>
      <div class="resize-handle resize-bottom" @mousedown="(e) => startResize(e, 'bottom')"></div>
      <div class="resize-handle resize-left" @mousedown="(e) => startResize(e, 'left')"></div>
      <div class="resize-handle resize-right" @mousedown="(e) => startResize(e, 'right')"></div>
      <div class="resize-handle resize-top-left" @mousedown="(e) => startResize(e, 'top-left')"></div>
      <div class="resize-handle resize-top-right" @mousedown="(e) => startResize(e, 'top-right')"></div>
      <div class="resize-handle resize-bottom-left" @mousedown="(e) => startResize(e, 'bottom-left')"></div>
      <div class="resize-handle resize-bottom-right" @mousedown="(e) => startResize(e, 'bottom-right')"></div>

      <div class="tabs-header" @mousedown="startDrag">
        <div class="tabs-container">
          <div v-for="(tab, index) in tabs" :key="tab.id" class="tab" :class="{ active: activeTabId === tab.id }" @click="activeTabId = tab.id">
            <span class="tab-icon">{{ tab.icon }}</span>
            <span class="tab-title">{{ tab.title }}</span>
            <button class="tab-close" @click.stop="closeTab(index)" title="Close tab">✕</button>
          </div>
        </div>
        <button class="btn-add-tab" @click="addNewTab" title="Add new editor">+ New</button>
      </div>
      <div class="editors-content">
        <div v-for="tab in tabs" :key="tab.id" class="editor-wrapper" :class="{ active: activeTabId === tab.id }">
          <div class="editor-header">
            <div class="editor-title">
              <span class="editor-icon">✏️</span>
              <span class="editor-resource">{{ tab.resourceType }} <strong>{{ tab.resourceName }}</strong></span>
              <span class="editor-namespace">in namespace <a href="#" class="ns-link">{{ tab.namespace }}</a></span>
            </div>
            <div class="editor-actions">
              <button class="btn btn-sm btn-save" @click="saveTab(tab.id)" title="Apply changes">💾 Save</button>
            </div>
          </div>
          <div :ref="(el) => setEditorContainer(tab.id, el)" class="editor-container"></div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, watch, onMounted, onBeforeUnmount } from "vue";
import * as monaco from "monaco-editor";

import "monaco-editor/esm/vs/editor/editor.main";
import "monaco-editor/esm/vs/basic-languages/yaml/yaml";
import codiconFont from "monaco-editor/esm/vs/base/browser/ui/codicons/codicon/codicon.ttf?url";

import { globalTabbedYamlEditor } from '../../composables/useTabbedYamlEditor';

const { tabs, activeTabId } = globalTabbedYamlEditor;

const emit = defineEmits<{
  (e: "update:content", val: string, tabId: string): void;
  (e: "apply", tabId: string): void;
  (e: "cancel", tabId: string): void;
  (e: "close"): void;
}>();

const panelHeight = ref(400);
const panelWidth = ref(800);
const panelLeft = ref(0);
const panelTop = ref(0);
const editors = new Map<string, monaco.editor.IStandaloneCodeEditor>();
const editorContainers = new Map<string, HTMLDivElement>();
let isResizing = false;
let isDragging = false;
let resizeDirection = "";
let dragStartX = 0;
let dragStartY = 0;
let dragStartLeft = 0;
let dragStartTop = 0;
let dragStartWidth = 0;
let dragStartHeight = 0;

onMounted(() => {
  document.documentElement.style.setProperty(
    "--codicon-font-url",
    `url('${codiconFont}')`
  );

  // Создаём редакторы для существующих табов
  tabs.value.forEach(tab => {
    setTimeout(() => createEditor(tab.id), 0);
  });

  if (tabs.value.length > 0) {
    activeTabId.value = tabs.value[0].id;
  } else {
    addNewTab();
  }

  window.addEventListener("mousemove", handleResize);
  window.addEventListener("mousemove", handleDrag);
  window.addEventListener("mouseup", stopResize);
  window.addEventListener("mouseup", stopDrag);
});

// Нет нужды в watch на props, так как состояние глобальное

function addNewTab() {
  globalTabbedYamlEditor.openTabbedYamlEditor("Resource", "Untitled", "default", "# New YAML document\n");
}

onBeforeUnmount(() => {
  window.removeEventListener("mousemove", handleResize);
  window.removeEventListener("mousemove", handleDrag);
  window.removeEventListener("mouseup", stopResize);
  window.removeEventListener("mouseup", stopDrag);
  editors.forEach((editor) => editor?.dispose());
});

// Нет локального addTab, используем глобальный openTabbedYamlEditor

function closeTab(index: number) {
  const tab = tabs.value[index];
  editors.delete(tab.id);
  editorContainers.delete(tab.id);
  tabs.value.splice(index, 1);

  if (tabs.value.length === 0) {
    emit("close");
  } else if (activeTabId.value === tab.id) {
    activeTabId.value = tabs.value[Math.max(0, index - 1)].id;
  }
}

function saveTab(tabId: string) {
  emit("apply", tabId);
  const tab = tabs.value.find((t) => t.id === tabId);
  if (tab) {
    tab.originalContent = tab.content;
  }
}

function cancelTab(tabId: string) {
  emit("cancel", tabId);
  const tab = tabs.value.find((t) => t.id === tabId);
  if (tab) {
    const editor = editors.get(tabId);
    if (editor) {
      editor.setValue(tab.originalContent);
    }
    tab.content = tab.originalContent;
  }
}

function setEditorContainer(tabId: string, el: any) {
  if (el) {
    editorContainers.set(tabId, el);
  }
}

function createEditor(tabId: string) {
  const tab = tabs.value.find((t) => t.id === tabId);
  if (!tab) return;

  const container = editorContainers.get(tabId);
  if (!container) return;

  const editor = monaco.editor.create(container, {
    value: tab.content,
    language: "yaml",
    theme: "vs-dark",
    automaticLayout: true,
    minimap: { enabled: true, side: "right" },
    scrollBeyondLastLine: false,
    wordWrap: "off",
    fontSize: 12,
    fontFamily: "'Monaco', 'Menlo', 'Ubuntu Mono', monospace",
    lineNumbers: "on",
    lineNumbersMinChars: 3,
    folding: true,
    formatOnPaste: false,
    formatOnType: false,
    glyphMargin: false,
    lineDecorationsWidth: 10,
    renderWhitespace: "none",
    padding: { top: 0, bottom: 0 },
    "bracketPairColorization.enabled": false,
    "semanticHighlighting.enabled": false,
    "editor.defaultColorDecorators": false,
    tabSize: 2,
    insertSpaces: true,
    readOnly: false,
    unicodeHighlight: {
      ambiguousCharacters: false,
      invisibleCharacters: false,
      nonBasicASCII: false,
    },
    "editor.renderLineHighlight": "none",
    "editor.occurrencesHighlight": "off",
    "editor.selectionHighlight": false,
  } as any);

  const model = editor.getModel();
  if (model) {
    monaco.editor.setModelMarkers(model, "owner", []);
    monaco.editor.setModelMarkers(model, "typescript", []);
    monaco.editor.setModelMarkers(model, "yaml", []);
    monaco.editor.setModelMarkers(model, "jsonc", []);
  }

  editor.onDidChangeModelContent(() => {
    const currentModel = editor.getModel();
    if (currentModel) {
      monaco.editor.setModelMarkers(currentModel, "owner", []);
      monaco.editor.setModelMarkers(currentModel, "typescript", []);
      monaco.editor.setModelMarkers(currentModel, "yaml", []);
    }

    const newValue = editor.getValue();
    const currentTab = tabs.value.find((t) => t.id === tabId);
    if (currentTab) {
      currentTab.content = newValue;
      emit("update:content", newValue, tabId);
    }
  });

  editors.set(tabId, editor);
}

function startResize(e: MouseEvent, direction: string) {
  isResizing = true;
  resizeDirection = direction;
  dragStartX = e.clientX;
  dragStartY = e.clientY;
  dragStartLeft = panelLeft.value;
  dragStartTop = panelTop.value;
  dragStartWidth = panelWidth.value;
  dragStartHeight = panelHeight.value;
  e.preventDefault();
}

function stopResize() {
  isResizing = false;
  resizeDirection = "";
}

function startDrag(e: MouseEvent) {
  isDragging = true;
  dragStartX = e.clientX;
  dragStartY = e.clientY;
  dragStartLeft = panelLeft.value;
  dragStartTop = panelTop.value;
}

function stopDrag() {
  isDragging = false;
}

function handleDrag(e: MouseEvent) {
  if (!isDragging) return;

  const deltaX = e.clientX - dragStartX;
  const deltaY = e.clientY - dragStartY;

  panelLeft.value = dragStartLeft + deltaX;
  panelTop.value = dragStartTop + deltaY;
}

function handleResize(e: MouseEvent) {
  if (!isResizing) return;

  const deltaX = e.clientX - dragStartX;
  const deltaY = e.clientY - dragStartY;

  if (resizeDirection.includes("top")) {
    panelTop.value = dragStartTop + deltaY;
    panelHeight.value = dragStartHeight - deltaY;
  }
  if (resizeDirection.includes("bottom")) {
    panelHeight.value = dragStartHeight + deltaY;
  }
  if (resizeDirection.includes("left")) {
    panelLeft.value = dragStartLeft + deltaX;
    panelWidth.value = dragStartWidth - deltaX;
  }
  if (resizeDirection.includes("right")) {
    panelWidth.value = dragStartWidth + deltaX;
  }

  // Минимальные размеры
  if (panelHeight.value < 100) panelHeight.value = 100;
  if (panelWidth.value < 300) panelWidth.value = 300;
}

// Watch на tabs для создания редакторов при добавлении новых табов
watch(tabs, (newTabs) => {
  newTabs.forEach(tab => {
    if (!editors.has(tab.id)) {
      setTimeout(() => createEditor(tab.id), 0);
    }
  });
}, { deep: true });

// Watch на activeTabId для layout
watch(activeTabId, () => {
  setTimeout(() => {
    const editor = editors.get(activeTabId.value);
    if (editor) {
      editor.layout();
    }
  }, 0);
});
</script>

<style scoped>
.tabbed-editor-wrapper {
  position: fixed;
  bottom: 0;
  right: 0;
  width: 100%;
  height: 100%;
  display: flex;
  flex-direction: column;
  z-index: 50;
  pointer-events: none;
}

.resize-handle {
  position: absolute;
  background: transparent;
  z-index: 100;
  pointer-events: auto;
  user-select: none;
}

.resize-top {
  top: 0;
  left: 0;
  right: 0;
  height: 6px;
  cursor: row-resize;
}

.resize-bottom {
  bottom: 0;
  left: 0;
  right: 0;
  height: 6px;
  cursor: row-resize;
}

.resize-left {
  left: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
}

.resize-right {
  right: 0;
  top: 0;
  bottom: 0;
  width: 6px;
  cursor: col-resize;
}

.resize-top-left {
  top: 0;
  left: 0;
  width: 10px;
  height: 10px;
  cursor: nwse-resize;
}

.resize-top-right {
  top: 0;
  right: 0;
  width: 10px;
  height: 10px;
  cursor: nesw-resize;
}

.resize-bottom-left {
  bottom: 0;
  left: 0;
  width: 10px;
  height: 10px;
  cursor: nesw-resize;
}

.resize-bottom-right {
  bottom: 0;
  right: 0;
  width: 10px;
  height: 10px;
  cursor: nwse-resize;
}

.resize-handle:hover {
  background: rgba(14, 99, 156, 0.5);
}

.tabbed-editor-panel {
  position: fixed;
  display: flex;
  flex-direction: column;
  background: #1e1e1e;
  color: #d4d4d4;
  border: 2px solid #0e639c;
  box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.5);
  flex-shrink: 0;
  min-height: 100px;
  pointer-events: auto;
  z-index: 1000;
  width: v-bind('panelWidth + "px"');
  height: v-bind('panelHeight + "px"');
  left: v-bind('panelLeft + "px"');
  top: v-bind('panelTop + "px"');
}

.tabs-header {
  display: flex;
  align-items: center;
  background: #252526;
  border-bottom: 1px solid #333;
  flex-shrink: 0;
  gap: 8px;
  padding: 0 8px;
  overflow-x: auto;
  overflow-y: hidden;
  height: 40px;
  cursor: grab;
}

.tabs-header:active {
  cursor: grabbing;
}

.tabs-container {
  display: flex;
  gap: 4px;
  flex: 1;
  min-width: 0;
}

.tab {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  background: #1e1e1e;
  border: 1px solid #3e3e42;
  border-bottom: none;
  border-radius: 4px 4px 0 0;
  cursor: pointer;
  color: #858585;
  font-size: 13px;
  white-space: nowrap;
  transition: all 0.15s;
  user-select: none;
}

.tab:hover {
  background: #2d2d30;
  color: #d4d4d4;
}

.tab.active {
  background: #1e1e1e;
  color: #d4d4d4;
  border-color: #0e639c;
  border-bottom-color: #1e1e1e;
}

.tab-icon {
  font-size: 14px;
  flex-shrink: 0;
}

.tab-title {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
}

.tab-close {
  padding: 0 4px;
  background: transparent;
  border: none;
  color: inherit;
  cursor: pointer;
  font-size: 12px;
  flex-shrink: 0;
  opacity: 0.6;
  transition: opacity 0.15s;
}

.tab-close:hover {
  opacity: 1;
  color: #ff6b6b;
}

.btn-add-tab {
  padding: 4px 8px;
  background: #0e639c;
  color: white;
  border: none;
  border-radius: 3px;
  cursor: pointer;
  font-size: 12px;
  margin-left: auto;
  flex-shrink: 0;
  transition: background 0.15s;
}

.btn-add-tab:hover {
  background: #1177bb;
}

.editors-content {
  flex: 1;
  overflow: hidden;
  position: relative;
}

.editor-wrapper {
  position: absolute;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  display: none;
  flex-direction: column;
  opacity: 0;
  transition: opacity 0.2s;
}

.editor-wrapper.active {
  display: flex;
  opacity: 1;
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
  padding: 6px 12px;
  border: none;
  border-radius: 4px;
  cursor: pointer;
  font-size: 12px;
  transition: all 0.15s;
}

.btn-sm {
  padding: 4px 8px;
  font-size: 11px;
}

.btn-save {
  background: #10b981;
  color: white;
}

.btn-save:hover {
  background: #059669;
}

.btn-secondary {
  background: #ef4444;
  color: white;
}

.btn-secondary:hover {
  background: #dc2626;
}

.editor-container {
  flex: 1;
  overflow: hidden;
  position: relative;
}
</style>
