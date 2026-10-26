<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="cl-overlay" @mousedown.self="pressedOverlay = true" @click.self="onOverlayClick">
    <div class="cl-dialog" role="dialog" aria-modal="true" :aria-labelledby="`${uid}-title`">
      <header class="cl-head">
        <ClusterAvatar :name="name" :icon="icon" :color="color" :environment="environment" :size="44" show-env />
        <div class="cl-head-text">
          <div :id="`${uid}-title`" class="cl-title">{{ name.trim() || 'Новый кластер' }}</div>
          <div class="cl-subtitle cl-mono">{{ subtitle }}</div>
        </div>
        <button type="button" class="cl-close" aria-label="Закрыть" @click="emit('close')">
          <svg viewBox="0 0 14 14" fill="none" stroke="currentColor" stroke-width="1.6" stroke-linecap="round">
            <path d="M2 2l10 10M12 2L2 12" />
          </svg>
        </button>
      </header>

      <div class="cl-body">
        <section class="cl-section">
          <h3 class="cl-section-title">Основное</h3>
          <div class="cl-field">
            <label :for="`${uid}-name`" class="cl-label">Название</label>
            <input :id="`${uid}-name`" ref="nameInput" v-model="name" class="cl-input" :maxlength="MAX_CLUSTER_NAME"
              placeholder="Например, Production EU" autocomplete="off" spellcheck="false"
              @keydown.enter.prevent="submit" />
          </div>
        </section>

        <section class="cl-section">
          <h3 class="cl-section-title">Подключение</h3>

          <div class="cl-field">
            <div class="cl-segment cl-method" role="radiogroup" aria-label="Источник kubeconfig">
              <button type="button" role="radio" class="cl-segment-item" :class="{ on: method === 'file' }"
                :aria-checked="method === 'file'" @click="setMethod('file')">Файл</button>
              <button type="button" role="radio" class="cl-segment-item" :class="{ on: method === 'text' }"
                :aria-checked="method === 'text'" @click="setMethod('text')">Вставить текст</button>
            </div>

            <div v-if="method === 'file'" class="cl-pick">
              <button type="button" class="cl-btn" @click="chooseFile">
                {{ filePath ? 'Выбрать другой…' : 'Выбрать kubeconfig…' }}
              </button>
              <span v-if="filePath" class="cl-pick-path cl-mono" :title="filePath">{{ filePath }}</span>
              <span v-else class="cl-hint" style="margin: 0">Файл не выбран</span>
            </div>
            <template v-else>
              <textarea v-model="text" class="cl-textarea cl-mono" spellcheck="false"
                placeholder="apiVersion: v1&#10;kind: Config&#10;…" aria-label="Текст kubeconfig"
                @input="onTextInput"></textarea>
              <p class="cl-hint">Содержимое будет сохранено в каталоге Kubron, права доступа — только для вас.</p>
            </template>

            <p v-if="hasSource && probe.inspectError.value" class="cl-hint is-error">{{ probe.inspectError.value }}</p>
          </div>

          <ClusterConnectionFields v-if="probe.info.value" v-model:context="contextName" v-model:namespace="namespace"
            :info="probe.info.value" :namespace-valid="namespaceValid" :check="probe.check.value"
            :checking="probe.checking.value" :can-test="!probe.inspecting.value" @test="runTest" />
        </section>

        <section class="cl-section">
          <h3 class="cl-section-title">Оформление</h3>
          <ClusterAppearanceFields v-model:icon="icon" v-model:color="color" v-model:environment="environment"
            :name="name" />
        </section>

        <div v-if="error" class="cl-error" role="alert">{{ error }}</div>
      </div>

      <footer class="cl-foot">
        <span class="cl-foot-spacer"></span>
        <button type="button" class="cl-btn" @click="emit('close')">Отмена</button>
        <button type="button" class="cl-btn cl-btn-primary" :disabled="!canSubmit" @click="submit">
          <span v-if="saving" class="cl-spin" aria-hidden="true"></span>
          Добавить кластер
        </button>
      </footer>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import type { AddClusterPayload, ClusterSettings, KubeconfigSource } from '../types'
import { NO_KUBECONFIG_SOURCE, pickContextName, useKubeconfigProbe } from '../composables/useKubeconfigProbe'
import ClusterAvatar from './cluster/ClusterAvatar.vue'
import ClusterAppearanceFields from './cluster/ClusterAppearanceFields.vue'
import ClusterConnectionFields from './cluster/ClusterConnectionFields.vue'
import { DEFAULT_CLUSTER_COLOR, MAX_CLUSTER_NAME, isValidNamespace } from './cluster/clusterStyle'

const props = defineProps<{
  pickFile: () => Promise<string>
  /** Возвращает текст ошибки или null, если кластер добавлен (диалог закрывает родитель) */
  submit: (payload: AddClusterPayload) => Promise<string | null>
}>()

const emit = defineEmits<{ (e: 'close'): void }>()

const uid = useId()
const probe = useKubeconfigProbe()

const name = ref('')
const icon = ref('')
const color = ref(DEFAULT_CLUSTER_COLOR)
const environment = ref('')
const namespace = ref('')
const contextName = ref('')

const method = ref<'file' | 'text'>('file')
const filePath = ref('')
const text = ref('')

const saving = ref(false)
const error = ref('')
const nameInput = ref<HTMLInputElement | null>(null)
const pressedOverlay = ref(false)

const hasSource = computed(() => (method.value === 'file' ? !!filePath.value : !!text.value.trim()))
const source = computed<KubeconfigSource>(() =>
  method.value === 'file'
    ? { ...NO_KUBECONFIG_SOURCE, path: filePath.value }
    : { ...NO_KUBECONFIG_SOURCE, text: text.value },
)

const namespaceValid = computed(() => isValidNamespace(namespace.value))
const canSubmit = computed(() =>
  !saving.value && name.value.trim().length > 0 && namespaceValid.value && hasSource.value &&
  !probe.inspecting.value && !probe.inspectError.value && !!probe.info.value,
)

const dirty = computed(() =>
  !!name.value || !!namespace.value || !!environment.value || hasSource.value,
)

const subtitle = computed(
  () => probe.info.value?.contexts.find(c => c.name === contextName.value)?.server || 'Добавление кластера в Kubron',
)

async function inspectSource() {
  probe.resetCheck()
  if (!hasSource.value) {
    probe.reset()
    return
  }
  const info = await probe.inspect(source.value)
  if (info) contextName.value = pickContextName(info, contextName.value)
}

async function chooseFile() {
  const path = await props.pickFile()
  if (!path) return
  error.value = ''
  filePath.value = path
  await inspectSource()
}

function setMethod(m: 'file' | 'text') {
  if (method.value === m) return
  method.value = m
  error.value = ''
  contextName.value = ''
  void inspectSource()
}

// Вставленный текст разбирается с задержкой — не на каждое нажатие клавиши
let inspectTimer: ReturnType<typeof setTimeout> | undefined
function onTextInput() {
  probe.resetCheck()
  clearTimeout(inspectTimer)
  inspectTimer = setTimeout(() => { void inspectSource() }, 450)
}

async function runTest() {
  await probe.test(source.value, contextName.value)
}

async function submit() {
  if (!canSubmit.value) return
  saving.value = true
  error.value = ''
  const settings: ClusterSettings = {
    displayName: name.value.trim(),
    icon: icon.value,
    color: color.value,
    environment: environment.value,
    defaultNamespace: namespace.value,
    contextName: contextName.value,
  }
  const payload: AddClusterPayload = method.value === 'file'
    ? { kind: 'file', path: filePath.value, settings }
    : { kind: 'text', text: text.value, settings }
  const err = await props.submit(payload)
  saving.value = false
  if (err) error.value = err
}

// Закрытие по клику мимо диалога — только пока форма пуста и нажатие началось на подложке
function onOverlayClick() {
  const startedOnOverlay = pressedOverlay.value
  pressedOverlay.value = false
  if (startedOnOverlay && !dirty.value && !saving.value) emit('close')
}

function onKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape' && !saving.value) {
    emit('close')
  } else if (e.key === 'Enter' && (e.ctrlKey || e.metaKey)) {
    void submit()
  }
}

onMounted(async () => {
  window.addEventListener('keydown', onKeydown)
  await nextTick()
  nameInput.value?.focus()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  clearTimeout(inspectTimer)
})
</script>
