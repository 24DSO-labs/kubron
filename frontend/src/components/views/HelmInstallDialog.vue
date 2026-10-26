<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="hi-overlay" @mousedown.self="pressedOverlay = true" @click.self="onOverlayClick">
    <div class="hi-dialog" role="dialog" aria-modal="true" aria-labelledby="hi-title">
      <header class="hi-head">
        <div>
          <div id="hi-title" class="hi-title">Установка {{ repo }}/{{ chart }}</div>
          <div class="hi-sub">версия {{ version }}</div>
        </div>
        <button type="button" class="hi-close" aria-label="Закрыть" :disabled="busy" @click="close">✕</button>
      </header>

      <div v-if="step === 'form'" class="hi-body">
        <div class="hi-fields">
          <label class="hi-field">Имя релиза
            <input v-model.trim="releaseName" class="form-input" :class="{ bad: nameBad }" maxlength="53" autocomplete="off"
              spellcheck="false" />
            <span v-if="nameBad" class="hi-hint bad">строчные латинские буквы, цифры и «-», до 53 символов</span>
          </label>
          <label class="hi-field">Namespace
            <input v-model.trim="namespace" class="form-input" :class="{ bad: nsBad }" list="hi-ns" autocomplete="off"
              spellcheck="false" />
            <datalist id="hi-ns"><option v-for="n in namespaceNames" :key="n" :value="n" /></datalist>
            <span v-if="nsBad" class="hi-hint bad">некорректный namespace</span>
          </label>
          <label class="hi-check">
            <input v-model="createNamespace" type="checkbox" /> Создать namespace, если его нет
            <span v-if="nsIsNew" class="hi-hint warn">— такого namespace в кластере нет</span>
          </label>
          <label class="hi-check">
            <input v-model="wait" type="checkbox" /> Дождаться готовности ресурсов (до 5 минут, окно блокируется)
          </label>
        </div>

        <div class="hi-editors">
          <section>
            <h4>Ваши значения (переопределения)</h4>
            <p class="hi-hint">Указывайте только то, что нужно изменить: остальное возьмётся из значений чарта справа.</p>
            <div class="hi-ed"><MonacoEditor v-model="overrides" language="yaml" /></div>
          </section>
          <section>
            <h4>Значения чарта по умолчанию</h4>
            <p class="hi-hint">Только для справки.</p>
            <div class="hi-ed"><MonacoReadonly :value="defaults || '# values.yaml недоступен'" language="yaml" /></div>
          </section>
        </div>
      </div>

      <div v-else class="hi-body">
        <div class="hi-seg" role="group" aria-label="Раздел">
          <button type="button" :class="{ on: previewTab === 'manifest' }" @click="previewTab = 'manifest'">Манифест</button>
          <button type="button" :class="{ on: previewTab === 'notes' }" @click="previewTab = 'notes'">Notes</button>
        </div>
        <p class="hi-hint">Предпросмотр без обращения к кластеру: lookup-функции шаблонов вернут пустое значение, значения Secret скрыты.</p>
        <div class="hi-ed hi-ed-tall">
          <MonacoReadonly v-if="previewTab === 'manifest'" :value="preview?.manifest || '# манифест пуст'" language="yaml" />
          <MonacoReadonly v-else :value="preview?.notes || '(у чарта нет NOTES.txt)'" language="plaintext" />
        </div>
      </div>

      <p v-if="error" class="form-error hi-error">{{ error }}</p>

      <footer class="hi-foot">
        <template v-if="step === 'form'">
          <button type="button" class="btn" :disabled="busy" @click="close">Отмена</button>
          <button type="button" class="btn" :disabled="!valid || busy" @click="doPreview">{{ busy ? 'Считаю…' : 'Предпросмотр' }}</button>
        </template>
        <template v-else>
          <button type="button" class="btn" :disabled="busy" @click="step = 'form'">Назад</button>
          <button type="button" class="btn hi-primary" :disabled="busy" @click="doInstall">
            {{ busy ? (wait ? 'Устанавливаю и жду готовности…' : 'Устанавливаю…') : 'Установить' }}
          </button>
        </template>
      </footer>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import MonacoEditor from '../resource/MonacoEditor.vue'
import MonacoReadonly from '../resource/MonacoReadonly.vue'
import { callApp } from '../../resources/wails'
import { useK8s } from '../../composables/useK8s'
import { useNotifications } from '../../composables/useNotifications'

interface InstallResult { releaseName: string; namespace: string; revision: number; status: string; manifest: string; notes: string }

const props = defineProps<{ repo: string; chart: string; version: string; defaults: string }>()
const emit = defineEmits<{ (e: 'close'): void; (e: 'installed', r: InstallResult): void }>()

const notifications = useNotifications()
const { namespaces, selectedNamespaces } = useK8s()

const DNS_LABEL = /^[a-z0-9]([-a-z0-9]{0,61}[a-z0-9])?$/
const RELEASE = /^[a-z0-9]([-a-z0-9]*[a-z0-9])?(\.[a-z0-9]([-a-z0-9]*[a-z0-9])?)*$/

const releaseName = ref(props.chart.toLowerCase().replace(/[^a-z0-9-]/g, '-').replace(/^-+|-+$/g, '').slice(0, 53))
const namespace = ref(selectedNamespaces.value[0] || 'default')
const createNamespace = ref(false)
const wait = ref(true)
const overrides = ref('')

const step = ref<'form' | 'preview'>('form')
const previewTab = ref<'manifest' | 'notes'>('manifest')
const preview = ref<InstallResult | null>(null)
const busy = ref(false)
const error = ref('')
const pressedOverlay = ref(false)

const namespaceNames = computed(() => (namespaces.value ?? []).map(n => n.name))
const nameBad = computed(() => releaseName.value !== '' && (releaseName.value.length > 53 || !RELEASE.test(releaseName.value)))
const nsBad = computed(() => namespace.value !== '' && !DNS_LABEL.test(namespace.value))
const nsIsNew = computed(() => DNS_LABEL.test(namespace.value) && !!namespaceNames.value.length && !namespaceNames.value.includes(namespace.value))
const valid = computed(() => !!releaseName.value && !!namespace.value && !nameBad.value && !nsBad.value)

function request() {
  return {
    repo: props.repo, chart: props.chart, version: props.version,
    releaseName: releaseName.value, namespace: namespace.value, values: overrides.value,
    createNamespace: createNamespace.value, wait: wait.value,
  }
}

async function doPreview() {
  busy.value = true
  error.value = ''
  try {
    preview.value = (await callApp('PreviewHelmInstall', request())) as InstallResult
    previewTab.value = 'manifest'
    step.value = 'preview'
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    busy.value = false
  }
}

async function doInstall() {
  busy.value = true
  error.value = ''
  try {
    const r = (await callApp('InstallHelmChart', request())) as InstallResult
    notifications.notifySuccess(`${r.releaseName}: установлен в ${r.namespace} (ревизия ${r.revision}). Релизы: Helm → Releases`)
    emit('installed', r)
    emit('close')
  } catch (e: any) {
    error.value = e?.message || String(e)
    step.value = 'form' // можно поправить имя или values и повторить
  } finally {
    busy.value = false
  }
}

function close() { if (!busy.value) emit('close') }
function onOverlayClick() {
  // закрываем по клику на фон только если и нажатие началось на фоне (не при выделении текста)
  if (pressedOverlay.value) close()
  pressedOverlay.value = false
}
function onKey(e: KeyboardEvent) { if (e.key === 'Escape') close() }
onMounted(() => window.addEventListener('keydown', onKey))
onBeforeUnmount(() => window.removeEventListener('keydown', onKey))
</script>

<style scoped>
.hi-overlay { position: fixed; inset: 0; z-index: 1000; background: rgba(0, 0, 0, 0.55); display: flex; align-items: center; justify-content: center; padding: 24px; }
.hi-dialog { width: min(1100px, 100%); max-height: 100%; display: flex; flex-direction: column; background: #1e1e2e; border: 1px solid #313244; border-radius: 10px; text-align: left; }
.hi-head { display: flex; justify-content: space-between; align-items: flex-start; padding: 16px 20px; border-bottom: 1px solid #313244; }
.hi-title { font-size: 18px; color: #89b4fa; font-weight: 600; }
.hi-sub { font-size: 12px; color: #a6adc8; margin-top: 2px; }
.hi-close { background: transparent; border: 0; color: #a6adc8; font-size: 16px; cursor: pointer; }
.hi-body { padding: 16px 20px; overflow: auto; display: flex; flex-direction: column; gap: 14px; min-height: 0; }
.hi-fields { display: grid; grid-template-columns: 1fr 1fr; gap: 12px 16px; }
.hi-field { display: flex; flex-direction: column; gap: 4px; font-size: 13px; color: #a6adc8; }
.hi-field .form-input { padding: 8px 10px; }
.form-input.bad { border-color: #f38ba8; }
.hi-check { grid-column: 1 / -1; display: flex; align-items: center; gap: 8px; font-size: 13px; color: #cdd6f4; }
.hi-hint { font-size: 12px; color: #6c7086; margin: 0; }
.hi-hint.bad { color: #f38ba8; }
.hi-hint.warn { color: #f9e2af; }
.hi-editors { display: grid; grid-template-columns: 1fr 1fr; gap: 16px; }
.hi-editors h4 { margin: 0 0 2px; font-size: 13px; color: #cdd6f4; }
.hi-ed { height: 36vh; min-height: 220px; margin-top: 6px; }
.hi-ed-tall { height: 50vh; }
.hi-ed :deep(.mr) { height: 100% !important; min-height: 0; }
.hi-seg { display: inline-flex; align-self: flex-start; border: 1px solid #3b3d5c; border-radius: 4px; overflow: hidden; }
.hi-seg button { padding: 4px 14px; background: transparent; color: #9ca3af; border: 0; font: inherit; cursor: pointer; }
.hi-seg button.on { background: #3b82f6; color: #fff; }
.hi-error { margin: 0 20px; white-space: pre-wrap; max-height: 20vh; overflow: auto; }
.hi-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 14px 20px; border-top: 1px solid #313244; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }
.hi-primary { background: #3b82f6; color: #fff; }
</style>
