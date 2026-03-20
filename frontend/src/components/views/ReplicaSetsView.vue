<template>
  <div>
    <ContentHeader icon="📊" title="ReplicaSets" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Ready</th><th>Images</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="rs in items" :key="rs.namespace+rs.name">
          <td class="resource-name">{{rs.name}}</td>
          <td class="ns-label">{{rs.namespace}}</td>
          <td><span :class="rs.ready===rs.desired?'text-green':'text-yellow'">{{rs.ready}}/{{rs.desired}}</span></td>
          <td class="images-cell"><div v-for="img in rs.images" :key="img" class="image-tag">{{img}}</div></td>
          <td>{{rs.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет ReplicaSets</div>
  </div>
</template>

<script lang="ts" setup>
import type { ReplicaSetInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: ReplicaSetInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>
