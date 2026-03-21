<template>
  <div>
    <ContentHeader icon="🔒" title="Network Policies" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Pod Selector</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="np in items" :key="np.namespace+np.name" class="clickable-row">
          <td class="resource-name">{{ np.name }}</td>
          <td class="ns-label">{{ np.namespace }}</td>
          <td class="mono">{{ np.podSelector }}</td>
          <td>{{ np.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Network Policies</div>
  </div>
</template>

<script lang="ts" setup>
import type { NetworkPolicyInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: NetworkPolicyInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
