<template>
  <div>
    <ContentHeader icon="📊" title="Resource Quotas" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>CPU Limit</th>
          <th>CPU Used</th>
          <th>Memory Limit</th>
          <th>Memory Used</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="rq in items" :key="rq.namespace+rq.name" class="clickable-row">
          <td class="resource-name">{{ rq.name }}</td>
          <td class="ns-label">{{ rq.namespace }}</td>
          <td class="mono">{{ rq.cpuLimit || '—' }}</td>
          <td class="mono">{{ rq.cpuUsed || '—' }}</td>
          <td class="mono">{{ rq.memLimit || '—' }}</td>
          <td class="mono">{{ rq.memUsed || '—' }}</td>
          <td>{{ rq.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет resource quotas</div>
  </div>
</template>

<script lang="ts" setup>
import type { ResourceQuotaInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: ResourceQuotaInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
