<template>
  <div>
    <ContentHeader icon="⚙️" title="Runtime Classes" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th>Name</th>
          <th>Handler</th>
          <th>Age</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="rc in items" :key="rc.name" class="clickable-row">
          <td class="resource-name">{{ rc.name }}</td>
          <td class="mono">{{ rc.handler }}</td>
          <td>{{ rc.age }}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Runtime Classes</div>
  </div>
</template>

<script lang="ts" setup>
import type { RuntimeClassInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

const props = defineProps<{
  items: RuntimeClassInfo[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>
