<template>
  <div>
    <ContentHeader icon="🌐" title="Services" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Type</th><th>Cluster IP</th><th>External IP</th><th>Ports</th><th>Age</th><th></th></tr></thead>
      <tbody>
        <tr v-for="svc in items" :key="svc.namespace+svc.name" @click="$emit('select', svc)" class="clickable-row">
          <td class="resource-name">{{ svc.name }}</td>
          <td class="ns-label">{{ svc.namespace }}</td>
          <td><span class="type-badge">{{ svc.type }}</span></td>
          <td>{{ svc.clusterIP }}</td>
          <td>{{ svc.externalIP || '—' }}</td>
          <td>{{ svc.ports?.join(', ') }}</td>
          <td>{{ svc.age }}</td>
          <td><button class="btn-icon" @click.stop="$emit('yaml', svc.namespace, svc.name)" title="YAML">📋</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Services</div>
  </div>
</template>

<script lang="ts" setup>
import type { ServiceInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

defineProps<{ items: ServiceInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'select', svc: ServiceInfo): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>