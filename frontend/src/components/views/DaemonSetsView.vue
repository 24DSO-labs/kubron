<template>
  <div>
    <ContentHeader icon="🔄" title="DaemonSets" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Desired</th><th>Current</th><th>Ready</th><th>Available</th><th>Images</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="ds in items" :key="ds.namespace+ds.name">
          <td class="resource-name">{{ds.name}}</td>
          <td class="ns-label">{{ds.namespace}}</td>
          <td>{{ds.desired}}</td>
          <td>{{ds.current}}</td>
          <td><span :class="ds.ready===ds.desired?'text-green':'text-yellow'">{{ds.ready}}</span></td>
          <td>{{ds.available}}</td>
          <td class="images-cell"><div v-for="img in ds.images" :key="img" class="image-tag">{{img}}</div></td>
          <td>{{ds.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет DaemonSets</div>
  </div>
</template>

<script lang="ts" setup>
import type { DaemonSetInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: DaemonSetInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>