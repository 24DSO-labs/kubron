<template>
  <div>
    <ContentHeader icon="⚙️" title="Jobs" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Completions</th><th>Active</th><th>Failed</th><th>Status</th><th>Duration</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="j in items" :key="j.namespace+j.name">
          <td class="resource-name">{{j.name}}</td>
          <td class="ns-label">{{j.namespace}}</td>
          <td>{{j.completions}}</td>
          <td>{{j.active}}</td>
          <td :class="{warn:j.failed>0}">{{j.failed}}</td>
          <td><span class="status-badge" :class="statusClass(j.status)">{{j.status}}</span></td>
          <td>{{j.duration}}</td>
          <td>{{j.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Jobs</div>
  </div>
</template>

<script lang="ts" setup>
import type { JobInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: JobInfo[]; search: string; statusClass: (s: string) => string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>