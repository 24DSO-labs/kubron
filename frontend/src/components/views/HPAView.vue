<template>
  <div>
    <ContentHeader icon="📊" title="Horizontal Pod Autoscalers" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Reference</th>
          <th>Min</th>
          <th>Max</th>
          <th>Current</th>
          <th>Desired</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="hpa in items" :key="hpa.namespace+hpa.name" class="clickable-row">
          <td class="resource-name">{{ hpa.name }}</td>
          <td class="ns-label">{{ hpa.namespace }}</td>
          <td class="mono">{{ hpa.reference }}</td>
          <td class="mono">{{ hpa.minReplicas }}</td>
          <td class="mono">{{ hpa.maxReplicas }}</td>
          <td class="mono">{{ hpa.currentReplicas }}</td>
          <td class="mono">{{ hpa.desiredReplicas }}</td>
          <td>{{ hpa.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет HPA</div>
  </div>
</template>

<script lang="ts" setup>
import type { HPAInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: HPAInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
