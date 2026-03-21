<template>
  <div>
    <ContentHeader icon="⭐" title="Priority Classes" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Value</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pc in items" :key="pc.name" class="clickable-row">
          <td class="resource-name">{{ pc.name }}</td>
          <td class="mono">{{ pc.value }}</td>
          <td>{{ pc.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Priority Classes</div>
  </div>
</template>

<script lang="ts" setup>
import type { PriorityClassInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: PriorityClassInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
