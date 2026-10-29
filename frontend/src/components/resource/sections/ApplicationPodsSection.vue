<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="rd-section">
    <div class="rd-section-title">{{ $t('defs.tabs.pods') }}</div>
    <div v-if="pods.length" class="rd-table-wrap">
      <table class="rd-table">
        <thead>
          <tr>
            <th>{{ $t('common.name') }}</th><th>{{ $t('defs.columns.node') }}</th><th>{{ $t('common.namespace') }}</th><th>{{ $t('defs.columns.ready') }}</th><th>{{ $t('defs.columns.cpu') }}</th><th>{{ $t('defs.columns.memory') }}</th><th>{{ $t('common.status') }}</th>
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
            <td class="rd-mono">{{ p.ready }}</td>
            <td class="rd-mono">{{ formatCpu(metrics[p.name]?.totalCpu) }}</td>
            <td class="rd-mono">{{ formatMemory(metrics[p.name]?.totalMem) }}</td>
            <td><span class="rd-badge" :class="badgeClass(p.status)">{{ p.status || $t('common.unknown') }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="rd-empty">{{ loading ? $t('common.loading') : $t('argo.pods.empty') }}</div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { applicationPods } from '../../../resources/applications'
import { badgeClass } from '../../../resources/status'
import { useK8s } from '../../../composables/useK8s'

/** Поды приложения (по labels app.kubernetes.io/instance + name) с текущей нагрузкой */
const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const k8s = useK8s()
const pods = ref<any[]>([])
const loading = ref(false)

async function load() {
  const app = props.resource
  if (!app?.namespace) { pods.value = []; return }
  loading.value = true
  try {
    const all = ((await callApp('GetPods', app.namespace)) as any[]) ?? []
    pods.value = applicationPods(all, app)
  } catch {
    pods.value = []
  } finally {
    loading.value = false
  }
}

// Метрики по имени пода (заполняет k8s.loadPodMetrics)
const metrics = computed(() => {
  const map: Record<string, any> = {}
  for (const m of k8s.podMetrics.value as any[]) map[m.name] = m
  return map
})
const { formatCpu, formatMemory } = k8s

watch(() => `${props.resource?.namespace}/${props.resource?.name}`, load, { immediate: true })
onMounted(() => { k8s.loadPodMetrics() })
</script>
