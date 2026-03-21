<template>
  <div>
    <ContentHeader icon="📏" title="Limit Ranges" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Items</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="lr in items" :key="lr.namespace+lr.name" class="clickable-row">
          <td class="resource-name">{{ lr.name }}</td>
          <td class="ns-label">{{ lr.namespace }}</td>
          <td class="mono">{{ lr.items }}</td>
          <td>{{ lr.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет limit ranges</div>
  </div>
</template>

<script lang="ts" setup>
import type { LimitRangeInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: LimitRangeInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
