<template>
  <div>
    <ContentHeader icon="🔄" title="Port Forwarding" :count="safeItems.length" @refresh="onRefresh" />
    <table v-if="safeItems.length > 0" class="table">
      <thead>
        <tr>
          <th>ID</th>
          <th>Namespace</th>
          <th>Pod</th>
          <th>Local Port</th>
          <th>Pod Port</th>
          <th>Status</th>
          <th>Action</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pf in safeItems" :key="pf.id" class="clickable-row">
          <td class="mono">{{ pf.id.substring(0, 8) }}...</td>
          <td class="ns-label">{{ pf.namespace }}</td>
          <td class="resource-name">{{ pf.pod }}</td>
          <td class="mono">{{ pf.localPort }}</td>
          <td class="mono">{{ pf.podPort }}</td>
          <td :class="statusClass(pf.status)">{{ pf.status }}</td>
          <td><button @click="onStop(pf.id)" class="btn-small">Stop</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет активных Port Forwarding сессий</div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { PortForwardSession } from '../../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: PortForwardSession[] | null
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'stop', id: string): void
}>()

const safeItems = computed(() => props.items || [])

function onRefresh() { emit('refresh') }
function onStop(id: string) { emit('stop', id) }

function statusClass(status: string): string {
  switch (status.toLowerCase()) {
    case 'active': case 'running': return 'status-running'
    case 'stopped': case 'closed': return 'status-error'
    default: return 'status-unknown'
  }
}
</script>
