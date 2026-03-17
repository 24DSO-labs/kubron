<template>
  <div>
    <ContentHeader icon="📎" title="PersistentVolumeClaims" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Status</th><th>Volume</th><th>Capacity</th><th>Access Modes</th><th>Storage Class</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="p in items" :key="p.namespace+p.name">
          <td class="resource-name">{{p.name}}</td>
          <td class="ns-label">{{p.namespace}}</td>
          <td><span class="status-badge" :class="statusClass(p.status)">{{p.status}}</span></td>
          <td>{{p.volume}}</td>
          <td>{{p.capacity}}</td>
          <td class="mono">{{p.accessModes}}</td>
          <td>{{p.storageClass}}</td>
          <td>{{p.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет PVC</div>
  </div>
</template>

<script lang="ts" setup>
import type { PVCInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: PVCInfo[]; search: string; statusClass: (s: string) => string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>