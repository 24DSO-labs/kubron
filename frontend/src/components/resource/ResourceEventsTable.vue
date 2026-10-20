<template>
  <div v-if="events.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr>
          <th>Type</th>
          <th>Reason</th>
          <th>Message</th>
          <th>Count</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ev in events" :key="ev.uid">
          <td><span class="rd-badge rd-badge-dot" :class="badgeClass(ev.type)"></span></td>
          <td class="rd-mono">{{ ev.reason }}</td>
          <td class="rd-muted">{{ ev.message }}</td>
          <td class="rd-mono">{{ ev.count }}</td>
          <td class="rd-mono">{{ ev.age || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">No events</div>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { GetEvents } from '../../../wailsjs/go/main/App'
import { timeAgo } from '../overview/format'
import { badgeClass } from '../../resources/status'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void }>()

interface EventRow {
  uid: string
  type: string
  reason: string
  message: string
  count: number
  age: string
}

const events = ref<EventRow[]>([])
let timer: ReturnType<typeof setInterval> | null = null
const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | null = null

async function load() {
  const ns = props.resource?.namespace ?? ''
  if (!ns && !props.resource?.name) return
  try {
    const all = ((await GetEvents(ns)) as any[]) ?? []
    const name = props.resource?.name ?? ''
    const kind = props.resource?._kind ?? ''
    const filtered = all.filter((e: any) => {
      // у кластерных ресурсов (PV, Node ...) ns пустой: события лежат в default, namespace не сверяем
      if (ns && e.namespace !== ns) return false
      if (kind && e.involvedKind !== kind) return false
      if (name && e.involvedName !== name) return false
      return true
    })
    events.value = filtered.map((e: any) => ({
      uid: e.name,                 // имя события уникально в namespace
      type: e.type || 'Normal',
      reason: e.reason || '',
      message: e.message || '',
      count: e.count || 1,         // у событий нового API count = 0
      age: e.age || '',            // Go уже отдаёт готовый age
    }))
  } catch {
    events.value = []
  }
}

onMounted(() => {
  load()
  timer = setInterval(load, 30_000)
  tick = setInterval(() => (now.value = Date.now()), 30_000)
})
onUnmounted(() => { if (timer) clearInterval(timer); if (tick) clearInterval(tick) })
</script>
