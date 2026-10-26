<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="cl-overlay" @mousedown.self="pressedOverlay = true" @click.self="onOverlayClick">
    <div class="cl-dialog" role="dialog" aria-modal="true" :aria-labelledby="`${uid}-title`">
      <header class="cl-head">
        <ClusterAvatar :name="name" :icon="icon" :color="color" :environment="environment" :size="44" show-env />
        <div class="cl-head-text">
          <div :id="`${uid}-title`" class="cl-title">{{ name.trim() || 'Без названия' }}</div>
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
              autocomplete="off" spellcheck="false" @keydown.enter.prevent="submit" />
          </div>
        </section>

        <section class="cl-section">
          <h3 class="cl-section-title">Оформление</h3>
          <ClusterAppearanceFields v-model:icon="icon" v-model:color="color" v-model:environment="environment"
            :name="name" />
        </section>

        <section class="cl-section">
          <h3 class="cl-section-title">Подключение</h3>

          <div class="cl-field">
            <span class="cl-label">Kubeconfig</span>
            <div class="cl-kc">
              <div class="cl-kc-info">
                <div class="cl-kc-name cl-mono" :title="kcLabel">{{ kcLabel }}</div>
                <div class="cl-kc-state" :class="{ 'is-changed': hasReplacement }">{{ kcState }}</div>
              </div>
              <div class="cl-kc-actions">
                <button type="button" class="cl-btn cl-btn-sm" @click="replaceWithFile">
                  {{ capabilities.fileDialog ? 'Заменить файлом…' : 'Загрузить из файла…' }}
                </button>
                <button v-if="kcMode !== 'text'" type="button" class="cl-btn cl-btn-sm" @click="editText">
                  Править текст
                </button>
                <button v-if="kcMode !== 'keep'" type="button" class="cl-btn cl-btn-sm cl-btn-quiet"
                  @click="revertKubeconfig">
                  Отменить
                </button>
              </div>
            </div>

            <template v-if="kcMode === 'text'">
              <textarea v-model="kcText" class="cl-textarea cl-mono" spellcheck="false" aria-label="Текст kubeconfig"
                style="margin-top: 8px" @input="onTextInput"></textarea>
              <p class="cl-hint is-warn">Текст содержит учётные данные кластера — не показывайте экран посторонним.</p>
            </template>
            <p v-if="hasReplacement && probe.inspectError.value" class="cl-hint is-error">
              {{ probe.inspectError.value }}
            </p>
          </div>

          <ClusterConnectionFields v-model:context="contextName" v-model:namespace="namespace"
            :info="probe.info.value" :namespace-valid="namespaceValid" :check="probe.check.value"
            :checking="probe.checking.value" :can-test="canTest" @test="runTest" />
        </section>

        <div v-if="error" class="cl-error" role="alert">{{ error }}</div>
      </div>

      <footer class="cl-foot">
        <button type="button" class="cl-btn cl-btn-danger" :disabled="saving" @click="emit('delete', cluster.id)">
          Удалить кластер
        </button>
        <span class="cl-foot-spacer"></span>
        <button type="button" class="cl-btn" @click="emit('close')">Отмена</button>
        <button type="button" class="cl-btn cl-btn-primary" :disabled="!canSave" @click="submit">
          <span v-if="saving" class="cl-spin" aria-hidden="true"></span>
          Сохранить
        </button>
      </footer>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, nextTick, onBeforeUnmount, onMounted, ref, useId } from 'vue'
import type { ClusterSettings, ClusterStatus, KubeconfigSource } from '../../types'
import { NO_KUBECONFIG_SOURCE, pickContextName, useKubeconfigProbe } from '../../composables/useKubeconfigProbe'
import { errorText } from '../../composables/errorText'
import ClusterAvatar from './ClusterAvatar.vue'
import ClusterAppearanceFields from './ClusterAppearanceFields.vue'
import ClusterConnectionFields from './ClusterConnectionFields.vue'
import { MAX_CLUSTER_NAME, findGlyph, isValidNamespace } from './clusterStyle'
import { callApp } from '../../platform/api'
import { pickTextFile } from '../../platform/files'
import { useCapabilities } from '../../composables/useCapabilities'

// Правка кластера: название, оформление, окружение и подключение (kubeconfig, контекст,
// namespace по умолчанию). Закрывает диалог родитель — после успешного сохранения.
const props = defineProps<{
  cluster: ClusterStatus
  pickFile: () => Promise<string>
  /** Возвращает текст ошибки или null, если сохранено */
  save: (id: string, settings: ClusterSettings, kubeconfig: KubeconfigSource) => Promise<string | null>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'delete', id: string): void
}>()

const uid = useId()
const probe = useKubeconfigProbe()

// Старый эмодзи-значок при первой правке заменяется монограммой
const initial = {
  name: props.cluster.displayName,
  icon: findGlyph(props.cluster.icon)?.key ?? '',
  color: props.cluster.color,
  environment: props.cluster.environment || '',
  namespace: props.cluster.defaultNamespace || '',
  context: props.cluster.contextName,
}

const name = ref(initial.name)
const icon = ref(initial.icon)
const color = ref(initial.color)
const environment = ref(initial.environment)
const namespace = ref(initial.namespace)
const contextName = ref(initial.context)

type KubeconfigMode = 'keep' | 'file' | 'text'
const { capabilities } = useCapabilities()
const kcMode = ref<KubeconfigMode>('keep')
const kcPath = ref('')
const kcText = ref('')
const kcOriginalText = ref('')

const saving = ref(false)
const error = ref('')
const nameInput = ref<HTMLInputElement | null>(null)
const pressedOverlay = ref(false)

const textDirty = computed(() => kcMode.value === 'text' && kcText.value !== kcOriginalText.value)
const hasReplacement = computed(() => (kcMode.value === 'file' && !!kcPath.value) || textDirty.value)

// Что отправить на сохранение: пустой источник означает «kubeconfig не менять»
const replacement = computed<KubeconfigSource>(() => {
  if (kcMode.value === 'file' && kcPath.value) return { ...NO_KUBECONFIG_SOURCE, path: kcPath.value }
  if (textDirty.value) return { ...NO_KUBECONFIG_SOURCE, text: kcText.value }
  return { ...NO_KUBECONFIG_SOURCE }
})

// Что разбирать и проверять: будущий kubeconfig, а без замены — сохранённый
const probeSource = computed<KubeconfigSource>(() =>
  hasReplacement.value ? replacement.value : { ...NO_KUBECONFIG_SOURCE, clusterId: props.cluster.id },
)

const nameValid = computed(() => name.value.trim().length > 0)
const namespaceValid = computed(() => isValidNamespace(namespace.value))
const canTest = computed(() => !!probe.info.value && !probe.inspecting.value)
const canSave = computed(() =>
  !saving.value && nameValid.value && namespaceValid.value &&
  !(hasReplacement.value && (probe.inspecting.value || !!probe.inspectError.value)),
)

const dirty = computed(() =>
  name.value !== initial.name || icon.value !== initial.icon || color.value !== initial.color ||
  environment.value !== initial.environment || namespace.value !== initial.namespace ||
  contextName.value !== initial.context || hasReplacement.value,
)

const selectedServer = computed(() => probe.info.value?.contexts.find(c => c.name === contextName.value)?.server ?? '')
const subtitle = computed(() => selectedServer.value || props.cluster.kubeconfigFile)

const kcLabel = computed(() => {
  if (kcMode.value === 'file' && kcPath.value) return kcPath.value
  return props.cluster.kubeconfigFile
})
const kcState = computed(() => {
  if (kcMode.value === 'file' && kcPath.value) return 'Будет заменён выбранным файлом'
  if (textDirty.value) return 'Изменён вручную — сохранится при нажатии «Сохранить»'
  if (kcMode.value === 'text') return 'Правка текста: пока без изменений'
  return 'Хранится в Kubron'
})

async function refreshProbe() {
  const info = await probe.inspect(probeSource.value)
  if (info) contextName.value = pickContextName(info, contextName.value)
}

async function replaceWithFile() {
  if (!capabilities.value.fileDialog) return replaceWithUpload()
  const path = await props.pickFile()
  if (!path) return
  error.value = ''
  kcMode.value = 'file'
  kcPath.value = path
  probe.resetCheck()
  await refreshProbe()
}

// Веб: файл читает браузер, на сервер уходит текст kubeconfig
async function replaceWithUpload() {
  try {
    const content = await pickTextFile()
    if (content === null) return
    error.value = ''
    kcOriginalText.value = ''
    kcText.value = content
    kcPath.value = ''
    kcMode.value = 'text'
    probe.resetCheck()
    await refreshProbe()
  } catch (e) {
    error.value = errorText(e)
  }
}

async function editText() {
  error.value = ''
  try {
    const text = await callApp('GetClusterKubeconfig', props.cluster.id)
    kcOriginalText.value = text
    kcText.value = text
    kcPath.value = ''
    kcMode.value = 'text'
  } catch (e) {
    error.value = errorText(e)
  }
}

async function revertKubeconfig() {
  kcMode.value = 'keep'
  kcPath.value = ''
  kcText.value = ''
  kcOriginalText.value = ''
  probe.resetCheck()
  await refreshProbe()
}

// Текст разбирается с задержкой — не на каждое нажатие клавиши
let inspectTimer: ReturnType<typeof setTimeout> | undefined
function onTextInput() {
  probe.resetCheck()
  clearTimeout(inspectTimer)
  inspectTimer = setTimeout(() => { void refreshProbe() }, 450)
}

async function runTest() {
  await probe.test(probeSource.value, contextName.value)
}

async function submit() {
  if (!canSave.value) return
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
  const err = await props.save(props.cluster.id, settings, replacement.value)
  saving.value = false
  if (err) error.value = err
}

// Закрытие по клику мимо диалога — только если ничего не изменено и нажатие началось на подложке:
// иначе неосторожный клик (или выделение мышью, вышедшее за край) стоил бы введённого текста
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
  nameInput.value?.select()
  await refreshProbe()
})

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown)
  clearTimeout(inspectTimer)
})
</script>
