<template>
  <div>
    <ContentHeader icon="🔌" title="Endpoints" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Addresses</th>
          <th>Ports</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ep in items" :key="ep.namespace+ep.name" class="clickable-row">
          <td class="resource-name">{{ ep.name }}</td>
          <td class="ns-label">{{ ep.namespace }}</td>
          <td class="mono">{{ ep.addresses }}</td>
          <td class="mono">{{ ep.ports }}</td>
          <td>{{ ep.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Endpoints</div>
  </div>
</template>

<script lang="ts" setup>
import type { EndpointListInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: EndpointListInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
