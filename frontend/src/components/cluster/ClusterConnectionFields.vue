<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div>
    <div v-if="info && info.contexts.length" class="cl-field">
      <label :for="`${uid}-ctx`" class="cl-label">{{ $t('cluster.connection.context') }}</label>
      <select :id="`${uid}-ctx`" class="cl-select" :value="context" @change="onContext">
        <option v-for="c in info.contexts" :key="c.name" :value="c.name">
          {{ c.name }}{{ c.name === info.currentContext ? $t('cluster.connection.defaultSuffix') : '' }}
        </option>
      </select>
      <p v-if="server" class="cl-hint cl-mono">{{ server }}</p>
    </div>

    <div class="cl-field">
      <label :for="`${uid}-ns`" class="cl-label">{{ $t('cluster.connection.namespace') }}</label>
      <input :id="`${uid}-ns`" class="cl-input cl-mono" :class="{ 'is-invalid': !namespaceValid }" :value="namespace"
        :placeholder="contextNamespace || $t('cluster.connection.allNamespaces')" spellcheck="false" autocomplete="off"
        @input="onNamespace" />
      <p v-if="!namespaceValid" class="cl-hint is-error">{{ $t('cluster.connection.namespaceInvalid') }}</p>
      <p v-else class="cl-hint">{{ $t('cluster.connection.namespaceAuto') }}</p>
    </div>

    <div class="cl-test">
      <button type="button" class="cl-btn" :disabled="!canTest || checking" @click="emit('test')">
        <span v-if="checking" class="cl-spin" aria-hidden="true"></span>
        {{ checking ? $t('cluster.connection.testing') : $t('cluster.connection.test') }}
      </button>
      <div class="cl-test-result" :class="resultClass" role="status" aria-live="polite">
        <template v-if="check?.ok && !check.warning">
          {{ $t('cluster.connection.reachable', { version: check.serverVersion, latency: check.latencyMs }) }}
        </template>
        <template v-else-if="check?.ok">
          {{ $t('cluster.connection.reachable', { version: check.serverVersion, latency: check.latencyMs }) }}
          <div class="cl-test-detail">{{ check.warning }}</div>
        </template>
        <template v-else-if="check">
          {{ $t('cluster.connection.failed') }}
          <div class="cl-test-detail cl-mono">{{ check.error }}</div>
        </template>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, useId } from 'vue'
import './cluster.css'
import type { ConnectionCheck, KubeconfigInfo } from '../../types'

// Контекст kubeconfig, namespace по умолчанию и проверка подключения: общая часть диалогов
// добавления и правки кластера.
const props = defineProps<{
  info: KubeconfigInfo | null
  context: string
  namespace: string
  namespaceValid: boolean
  check: ConnectionCheck | null
  checking: boolean
  /** Есть что проверять: выбран и разобран kubeconfig */
  canTest: boolean
}>()

const emit = defineEmits<{
  (e: 'update:context', v: string): void
  (e: 'update:namespace', v: string): void
  (e: 'test'): void
}>()

const uid = useId()

const selected = computed(() => props.info?.contexts.find(c => c.name === props.context) ?? null)
const server = computed(() => selected.value?.server ?? '')
// Namespace, заданный самим контекстом kubeconfig, показывается подсказкой
const contextNamespace = computed(() => selected.value?.namespace ?? '')

const resultClass = computed(() => {
  if (!props.check) return ''
  if (!props.check.ok) return 'is-fail'
  return props.check.warning ? 'is-warn' : 'is-ok'
})

function onNamespace(e: Event) {
  emit('update:namespace', (e.target as HTMLInputElement).value.trim())
}

function onContext(e: Event) {
  emit('update:context', (e.target as HTMLSelectElement).value)
}
</script>
