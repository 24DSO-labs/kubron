<template>
  <div>
    <ContentHeader icon="📚" title="StatefulSets" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Ready</th><th>Service</th><th>Images</th><th>Age</th></tr></thead>
      <tbody>
        <tr v-for="ss in items" :key="ss.namespace+ss.name">
          <td class="resource-name">{{ss.name}}</td>
          <td class="ns-label">{{ss.namespace}}</td>
          <td><span :class="ss.readyReplicas===ss.replicas?'text-green':'text-yellow'">{{ss.readyReplicas}}/{{ss.replicas}}</span></td>
          <td>{{ss.serviceName}}</td>
          <td class="images-cell"><div v-for="img in ss.images" :key="img" class="image-tag">{{img}}</div></td>
          <td>{{ss.age}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет StatefulSets</div>
  </div>
</template>

<script lang="ts" setup>
import type { StatefulSetInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: StatefulSetInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>