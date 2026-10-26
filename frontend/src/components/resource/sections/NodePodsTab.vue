<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="rows.length" class="rd-table-wrap">
    <table class="rd-table np-table">
      <thead>
        <tr>
          <th>Name</th><th>Node</th><th>Namespace</th><th>Ready</th><th>CPU</th><th>Memory</th><th>Status</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="p in rows"
          :key="`${p.namespace}/${p.name}`"
          class="rd-click"
          @click="emit('open', { id: 'pod', ns: p.namespace, name: p.name, item: p })"
        >
          <td class="rd-link np-trunc" :title="p.name">{{ p.name }}</td>
          <td class="np-trunc" :title="p.node">{{ p.node }}</td>
          <td class="rd-muted np-trunc" :title="p.namespace">{{ p.namespace }}</td>
          <td class="rd-mono">{{ readyText(p.ready) }}</td>
          <td>
            <div class="np-bar" :title="cpuTip(p)">
              <div class="np-fill np-cpu" :style="{ width: cpuPct(p) + '%' }"></div>
            </div>
          </td>
          <td>
            <div class="np-bar" :title="memTip(p)">
              <div class="np-fill np-mem" :style="{ width: memPct(p) + '%' }"></div>
            </div>
          </td>
          <td :style="{ color: statusColor(p.status) }">{{ p.status || 'Unknown' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ loading ? 'Загрузка...' : 'No pods on this node' }}</div>
</template>

<script lang="ts" setup>
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { statusColor } from '../../../resources/status'
import { useK8s } from '../../../composables/useK8s'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const REFRESH_MS = 15_000
const k8s = useK8s()

const pods = ref<any[]>([])
const metrics = ref<Record<string, any>>({})
const loading = ref(true)

const nodeName = computed(() => String(props.resource?.name ?? ''))
const nodeCores = computed(() => Number(props.resource?.cpuCores) || 0)
const nodeMemory = computed(() => Number(props.resource?.memoryBytes) || 0)

const rows = computed(() =>
  [...pods.value].sort((a, b) => a.namespace.localeCompare(b.namespace) || a.name.localeCompare(b.name)),
)

let seq = 0
async function load() {
  const my = ++seq
  const name = nodeName.value
  if (!name) return
  // Метрики опциональны (стек может быть не установлен): их отказ не должен прятать список
  const [p, m] = await Promise.allSettled([callApp('GetPods', []), callApp('GetPodMetrics', [])])
  if (my !== seq) return
  if (p.status === 'fulfilled') pods.value = ((p.value as any[]) ?? []).filter(x => x.node === name)
  if (m.status === 'fulfilled') {
    const map: Record<string, any> = {}
    for (const x of (m.value as any[]) ?? []) map[`${x.namespace}/${x.name}`] = x
    metrics.value = map
  }
  loading.value = false
}

// ─── Использование ───
const keyOf = (p: any) => `${p.namespace}/${p.name}`
const clamp = (v: number) => Math.min(Math.max(v, 0), 100)

/** CPU пода в милликорах (GetPodMetrics отдаёт rate * 1000) */
function cpuUsed(p: any): number | null {
  const v = parseFloat(metrics.value[keyOf(p)]?.totalCpu)
  return isNaN(v) ? null : v
}
/** Память пода в байтах */
function memUsed(p: any): number | null {
  const v = parseFloat(metrics.value[keyOf(p)]?.totalMem)
  return isNaN(v) ? null : v
}

function cpuPct(p: any): number {
  const u = cpuUsed(p)
  return u === null || !nodeCores.value ? 0 : clamp((u / (nodeCores.value * 1000)) * 100)
}
function memPct(p: any): number {
  const u = memUsed(p)
  return u === null || !nodeMemory.value ? 0 : clamp((u / nodeMemory.value) * 100)
}

// Тулчейн: значение и доля от ёмкости узла
function cpuTip(p: any): string {
  const u = cpuUsed(p)
  return u === null ? 'CPU: no metrics' : `CPU: ${k8s.formatCpu(String(u))} (${cpuPct(p).toFixed(2)}% of node)`
}
function memTip(p: any): string {
  const u = memUsed(p)
  return u === null ? 'Memory: no metrics' : `Memory: ${k8s.formatMemory(String(u))} (${memPct(p).toFixed(2)}% of node)`
}

const readyText = (r?: string) => (r ?? '').replace('/', ' / ')

let timer: ReturnType<typeof setInterval> | null = null
onMounted(() => {
  load()
  timer = setInterval(() => { if (!document.hidden) load() }, REFRESH_MS)
})
onUnmounted(() => {
  seq++
  if (timer) clearInterval(timer)
})
watch(nodeName, () => { pods.value = []; loading.value = true; load() })
</script>

<style scoped>
.np-table { table-layout: fixed; }
.np-trunc { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.np-table th:nth-child(1) { width: 22%; }
.np-table th:nth-child(4) { width: 64px; }
.np-table th:nth-child(5), .np-table th:nth-child(6) { width: 100px; }
.np-table th:nth-child(7) { width: 84px; }

.np-bar { width: 74px; height: 4px; background: #2f3040; border-radius: 2px; overflow: hidden; cursor: help; }
.np-fill { height: 100%; border-radius: 2px; min-width: 2px; transition: width 0.4s ease; }
.np-cpu { background: #3b9edb; }
.np-mem { background: #c04fc9; }
</style>
