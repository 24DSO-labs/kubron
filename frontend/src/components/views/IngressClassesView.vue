<template>
  <div>
    <ContentHeader icon="🎯" title="Ingress Classes" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Controller</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="ic in items" :key="ic.name" class="clickable-row">
          <td class="resource-name">{{ ic.name }}</td>
          <td class="mono">{{ ic.controller }}</td>
          <td>{{ ic.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Ingress Classes</div>
  </div>
</template>

<script lang="ts" setup>
import type { IngressClassInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: IngressClassInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
