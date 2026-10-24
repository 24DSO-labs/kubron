<template>
  <div v-if="history.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>Revision</th><th>Status</th><th>Chart</th><th>App version</th><th>Updated</th><th>Description</th></tr>
      </thead>
      <tbody>
        <tr v-for="h in history" :key="h.revision">
          <td class="rd-mono">#{{ h.revision }}<span v-if="h.revision === current" class="rd-muted"> (current)</span></td>
          <td><span class="rd-badge" :class="badgeClass(h.status)">{{ h.status }}</span></td>
          <td class="rd-mono">{{ h.chart }}-{{ h.chartVersion }}</td>
          <td class="rd-mono">{{ h.appVersion || '—' }}</td>
          <td class="rd-muted" :title="h.updatedAt">{{ h.updated || '—' }}</td>
          <td class="rd-muted">{{ h.description || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">No history</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { badgeClass } from '../../../resources/status'

const props = defineProps<{ resource: any }>()
const history = computed<any[]>(() => props.resource?.history ?? [])
const current = computed(() => Number(props.resource?.revision) || 0)
</script>