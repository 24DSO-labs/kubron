<template>
  <div>
    <ContentHeader icon="🔒" title="Secrets" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Type</th><th>Data Keys</th><th>Age</th><th></th></tr></thead>
      <tbody>
        <tr v-for="s in items" :key="s.namespace+s.name">
          <td class="resource-name">{{s.name}}</td>
          <td class="ns-label">{{s.namespace}}</td>
          <td><span class="type-badge">{{s.type}}</span></td>
          <td>{{s.dataCount}}</td>
          <td>{{s.age}}</td>
          <td><button class="btn-icon" @click="$emit('yaml',s.namespace,s.name)">📋</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Secrets</div>
  </div>
</template>

<script lang="ts" setup>
import type { SecretInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: SecretInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void; (e: 'yaml', ns: string, name: string): void }>()
</script>