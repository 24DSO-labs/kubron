<template>
  <div>
    <ContentHeader icon="🛡️" title="Pod Disruption Budgets" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Namespace</th>
          <th>Min Available</th>
          <th>Max Unavailable</th>
          <th>Disruptions Allowed</th>
          <th>Current Healthy</th>
          <th>Desired Healthy</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="pdb in items" :key="pdb.namespace+pdb.name" class="clickable-row">
          <td class="resource-name">{{ pdb.name }}</td>
          <td class="ns-label">{{ pdb.namespace }}</td>
          <td class="mono">{{ pdb.minAvailable }}</td>
          <td class="mono">{{ pdb.maxUnavailable }}</td>
          <td class="mono">{{ pdb.disruptionsAllowed }}</td>
          <td class="mono">{{ pdb.currentHealthy }}</td>
          <td class="mono">{{ pdb.desiredHealthy }}</td>
          <td>{{ pdb.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Pod Disruption Budgets</div>
  </div>
</template>

<script lang="ts" setup>
import type { PDBInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: PDBInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
