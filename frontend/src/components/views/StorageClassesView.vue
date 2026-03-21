<template>
  <div>
    <ContentHeader
      icon="💾"
      title="Storage Classes"
      :count="items.length"
      :search="search"
      showSearch
      @refresh="onRefresh"
      @search="onSearch"
    />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Provisioner</th>
          <th>Reclaim Policy</th>
          <th>Default</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="sc in items" :key="sc.name">
          <td class="resource-name">{{ sc.name }}</td>
          <td>{{ sc.provisioner }}</td>
          <td>{{ sc.reclaimPolicy }}</td>
          <td><span :class="sc.default ? 'text-green' : 'text-gray'">{{ sc.default ? 'Yes' : 'No' }}</span></td>
          <td>{{ sc.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Storage Classes</div>
  </div>
</template>

<script lang="ts" setup>
import type { StorageClassInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

defineProps<{
  items: StorageClassInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
