<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="pods.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr>
          <th>{{ $t('common.name') }}</th><th>{{ $t('defs.columns.node') }}</th><th>{{ $t('common.namespace') }}</th><th>{{ $t('defs.columns.controller') }}</th><th>{{ $t('defs.columns.ready') }}</th><th>{{ $t('common.status') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in pods"
          :key="`${p.namespace}/${p.name}`"
          class="rd-click"
          @click="emit('open', { id: 'pod', ns: p.namespace, name: p.name, item: p })"
        >
          <td class="rd-link">{{ p.name }}</td>
          <td>{{ p.node || '—' }}</td>
          <td class="rd-muted">{{ p.namespace }}</td>
          <td class="rd-muted">{{ p.controlledBy || '—' }}</td>
          <td class="rd-mono">{{ p.ready }}</td>
          <td><span class="rd-badge" :class="badgeClass(p.status)">{{ p.status || $t('common.unknown') }}</span></td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ loading ? $t('common.loading') : $t('sections.podsByClass.empty') }}</div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { badgeClass } from '../../../resources/status'
import { itemName } from '../../../resources/utils'

/** field — поле pod, по которому ищем (priorityClassName / runtimeClassName); значение сравнивается с именем ресурса */
const props = defineProps<{ resource: any; field: 'priorityClassName' | 'runtimeClassName' }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const pods = ref<any[]>([])
const loading = ref(false)

/** Поды всех namespace, у которых поле класса совпадает с именем ресурса (PriorityClass и RuntimeClass глобальные) */
async function load() {
  const name = props.resource ? itemName(props.resource) : ''
  if (!name) { pods.value = []; return }
  loading.value = true
  try {
    const all = ((await callApp('GetPods', [])) as any[]) ?? []
    // Ответ мог устареть, если за время запроса открыли другой класс
    if (name === itemName(props.resource)) pods.value = all.filter(p => p[props.field] === name)
  } catch {
    pods.value = []
  } finally {
    loading.value = false
  }
}

watch(() => itemName(props.resource), load, { immediate: true })
</script>
