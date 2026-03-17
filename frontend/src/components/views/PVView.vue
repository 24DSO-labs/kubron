<template>
  <div>
    <ContentHeader icon="💾" title="PersistentVolumes" :count="items.length" @refresh="$emit('refresh')" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Capacity</th><th>Access Modes</th><th>Reclaim Policy</th><th>Status</th><th>Claim</th><th>Storage Class</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="p in items" :key="p.name">
          <td class="resource-name">{{p.name}}</td>
          <td>{{p.capacity}}</td>
          <td class="mono">{{p.accessModes}}</td>
          <td>{{p.reclaimPolicy}}</td>
          <td><span class="status-badge" :class="statusClass(p.status)">{{p.status}}</span></td>
          <td class="ns-label">{{p.claim}}</td>
          <td>{{p.storageClass}}</td>
          <td>{{p.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет PersistentVolumes</div>
  </div>
</template>

<script lang="ts" setup>
import type { PVInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: PVInfo[]; statusClass: (s: string) => string }>()
defineEmits<{ (e: 'refresh'): void }>()
</script>