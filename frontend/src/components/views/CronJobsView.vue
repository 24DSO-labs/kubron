<template>
  <div>
    <ContentHeader
      icon="🕐"
      title="CronJobs"
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
          <th>Namespace</th>
          <th>Schedule</th>
          <th>Suspend</th>
          <th>Active</th>
          <th>Last Schedule</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="cj in items" :key="cj.namespace + cj.name">
          <td class="resource-name">{{ cj.name }}</td>
          <td class="ns-label">{{ cj.namespace }}</td>
          <td class="mono">{{ cj.schedule }}</td>
          <td><span :class="cj.suspend ? 'text-yellow' : 'text-green'">{{ cj.suspend ? 'Yes' : 'No' }}</span></td>
          <td>{{ cj.active }}</td>
          <td>{{ cj.lastSchedule }}</td>
          <td>{{ cj.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет CronJobs</div>
  </div>
</template>

<script lang="ts" setup>
import type { CronJobInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

defineProps<{
  items: CronJobInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>