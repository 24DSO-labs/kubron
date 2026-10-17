<template>
  <section class="ov-card workloads">
    <header class="head">
      <h2 class="ov-card-title">Workloads</h2>
      <span class="count">{{ filtered.length }}<template v-if="filtered.length !== workloads.length"> of {{ workloads.length }}</template></span>

      <div class="tools">
        <select v-model="namespace" class="ov-input" aria-label="Namespace">
          <option value="">All namespaces</option>
          <option v-for="ns in namespaces" :key="ns" :value="ns">{{ ns }}</option>
        </select>
        <input v-model="search" class="ov-input" type="search" placeholder="Search workloads…" aria-label="Search workloads" />
      </div>
    </header>

    <div class="scroll">
      <table v-if="filtered.length" class="table">
        <thead>
          <tr>
            <th>Name</th>
            <th>Namespace</th>
            <th>Type</th>
            <th>Status</th>
            <th>Ready</th>
            <th>Up to date</th>
            <th>Age</th>
          </tr>
        </thead>
        <tbody>
          <tr
            v-for="w in filtered"
            :key="`${w.kind}/${w.namespace}/${w.name}`"
            tabindex="0"
            @click="emit('open', w)"
            @keydown.enter="emit('open', w)"
          >
            <td class="name">{{ w.name }}</td>
            <td class="muted">{{ w.namespace }}</td>
            <td>{{ w.kind }}</td>
            <td>
              <span class="pill" :class="statusClass(w.status)">
                <span class="pill-dot"></span>{{ w.status }}
              </span>
            </td>
            <td class="num">{{ w.ready }}/{{ w.desired }}</td>
            <td class="num">{{ w.upToDate }}</td>
            <td class="muted">{{ w.age }}</td>
          </tr>
        </tbody>
      </table>
      <p v-else class="none">{{ workloads.length ? 'Nothing matches the filter' : 'No workloads in this cluster' }}</p>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import type { WorkloadSummary } from '../../types/overview'

const props = defineProps<{ workloads: WorkloadSummary[] }>()
const emit = defineEmits<{ (e: 'open', w: WorkloadSummary): void }>()

const namespace = ref('')
const search = ref('')

const namespaces = computed(() => [...new Set(props.workloads.map(w => w.namespace))].sort())

// namespace мог исчезнуть из кластера (или мы переключили кластер): сбрасываем фильтр
watch(namespaces, list => {
  if (namespace.value && !list.includes(namespace.value)) namespace.value = ''
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return props.workloads.filter(
    w => (!namespace.value || w.namespace === namespace.value) && (!q || w.name.toLowerCase().includes(q)),
  )
})

function statusClass(status: WorkloadSummary['status']) {
  switch (status) {
    case 'Running': return 'p-ok'
    case 'Degraded': return 'p-warn'
    case 'Unavailable': return 'p-danger'
    default: return 'p-idle'
  }
}
</script>

<style scoped>
.workloads { min-width: 0; }
.head { display: flex; align-items: center; gap: 12px; margin-bottom: 12px; flex-wrap: wrap; }
.head .ov-card-title { margin: 0; }
.count { font-size: 12px; color: var(--ov-muted); }
.tools { margin-left: auto; display: flex; gap: 8px; flex-wrap: wrap; }
.tools select { min-width: 150px; }
.tools input { width: 190px; }

.scroll { overflow: auto; max-height: 330px; margin: 0 -4px; padding: 0 4px; }

.table { width: 100%; border-collapse: collapse; font-size: 13px; }
.table th {
  position: sticky;
  top: 0;
  z-index: 1;
  padding: 8px 10px;
  text-align: left;
  font-weight: 500;
  font-size: 12px;
  color: var(--ov-muted);
  background: var(--ov-card);
  border-bottom: 1px solid var(--ov-border);
  white-space: nowrap;
}
.table td { padding: 9px 10px; border-bottom: 1px solid var(--ov-border-soft); white-space: nowrap; text-align: left; }
.table tbody tr { cursor: pointer; }
.table tbody tr:hover, .table tbody tr:focus-visible { background: var(--ov-card-hover); outline: none; }
.table tbody tr:last-child td { border-bottom: none; }
.name { font-weight: 500; max-width: 260px; overflow: hidden; text-overflow: ellipsis; }
.muted { color: var(--ov-muted); }
.num { font-variant-numeric: tabular-nums; }

.pill {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 2px 9px;
  border-radius: 999px;
  font-size: 12px;
  font-weight: 500;
}
.pill-dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.p-ok { color: var(--ov-ok); background: color-mix(in srgb, var(--ov-ok) 14%, transparent); }
.p-warn { color: var(--ov-warn); background: color-mix(in srgb, var(--ov-warn) 14%, transparent); }
.p-danger { color: var(--ov-danger); background: color-mix(in srgb, var(--ov-danger) 14%, transparent); }
.p-idle { color: var(--ov-muted); background: color-mix(in srgb, var(--ov-muted) 14%, transparent); }

.none { padding: 24px 0; text-align: center; font-size: 13px; color: var(--ov-muted); }
</style>
