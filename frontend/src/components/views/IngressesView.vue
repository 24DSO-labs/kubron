<template>
  <div>
    <ContentHeader icon="🔀" title="Ingresses" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Class</th><th>Hosts</th><th>Paths</th><th>Address</th><th>TLS</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="ing in items" :key="ing.namespace+ing.name">
          <td class="resource-name">{{ing.name}}</td>
          <td class="ns-label">{{ing.namespace}}</td>
          <td>{{ing.class||'—'}}</td>
          <td><div v-for="h in ing.hosts" :key="h" class="host-tag">{{h}}</div></td>
          <td class="paths-cell">{{ing.paths}}</td>
          <td>{{ing.address||'—'}}</td>
          <td><span :class="ing.tls?'text-green':'text-yellow'">{{ing.tls?'🔒 Yes':'No'}}</span></td>
          <td>{{ing.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Ingresses</div>
  </div>
</template>

<script lang="ts" setup>
import type { IngressInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: IngressInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>