<template>
  <div v-if="pods.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr>
          <th>Name</th><th>Node</th><th>Namespace</th><th>Controller</th><th>Ready</th><th>Status</th>
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
          <td><span class="rd-badge" :class="badgeClass(p.status)">{{ p.status || 'Unknown' }}</span></td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ loading ? 'Загрузка...' : 'No pods use this priority class' }}</div>
</template>

<script lang="ts" setup>
import { ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { badgeClass } from '../../../resources/status'
import { itemName } from '../../../resources/utils'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const pods = ref<any[]>([])
const loading = ref(false)

/** Поды всех namespace, у которых spec.priorityClassName совпадает с именем класса (PriorityClass глобальный) */
async function load() {
  const name = props.resource ? itemName(props.resource) : ''
  if (!name) { pods.value = []; return }
  loading.value = true
  try {
    const all = ((await callApp('GetPods', '')) as any[]) ?? []
    // Ответ мог устареть, если за время запроса открыли другой класс
    if (name === itemName(props.resource)) pods.value = all.filter(p => p.priorityClassName === name)
  } catch {
    pods.value = []
  } finally {
    loading.value = false
  }
}

watch(() => itemName(props.resource), load, { immediate: true })
</script>
