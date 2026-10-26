<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="pods.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr>
          <th>Name</th><th>Node</th><th>Namespace</th><th>Controller</th><th>Ready</th><th>CPU</th><th>Memory</th><th>Status</th>
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
          <td class="rd-mono">{{ formatCpu(metrics[p.name]?.totalCpu) }}</td>
          <td class="rd-mono">{{ formatMemory(metrics[p.name]?.totalMem) }}</td>
          <td><span class="rd-badge" :class="badgeClass(p.status)">{{ p.status || 'Unknown' }}</span></td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ loading ? 'Загрузка...' : 'No pods available' }}</div>
</template>

<script lang="ts" setup>
import { onMounted } from 'vue'
import { useControlledPods } from '../../../resources/useControlledPods'
import { badgeClass } from '../../../resources/status'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const { pods, loading, metrics, formatCpu, formatMemory, loadMetrics } = useControlledPods(() => props.resource)

// Метрики иначе заполняются только на view "Pods"
onMounted(() => { loadMetrics() })
</script>
