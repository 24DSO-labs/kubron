<template>
  <div>
    <ContentHeader icon="🔑" title="Leases" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Holder</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="lease in items" :key="lease.namespace+lease.name" class="clickable-row">
          <td class="resource-name">{{ lease.name }}</td>
          <td class="ns-label">{{ lease.namespace }}</td>
          <td class="mono">{{ lease.holder }}</td>
          <td>{{ lease.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Leases</div>
  </div>
</template>

<script lang="ts" setup>
import type { LeaseInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: LeaseInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
