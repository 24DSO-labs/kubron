<template>
  <div>
    <ContentHeader icon="📄" title="ConfigMaps" :count="items.length" :search="search" showSearch @refresh="$emit('refresh')" @search="$emit('search', $event)" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Data Keys</th><th>Age</th><th></th></tr></thead>
      <tbody>
        <tr v-for="cm in items" :key="cm.namespace+cm.name" @click="$emit('select',cm)" class="clickable-row">
          <td class="resource-name">{{cm.name}}</td>
          <td class="ns-label">{{cm.namespace}}</td>
          <td>{{cm.dataCount}}</td>
          <td>{{cm.age}}</td>
          <td><button class="btn-icon" @click.stop="$emit('yaml',cm.namespace,cm.name)">📋</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет ConfigMaps</div>
  </div>
</template>

<script lang="ts" setup>
import type { ConfigMapInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: ConfigMapInfo[]; search: string }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void; (e: 'select', cm: ConfigMapInfo): void; (e: 'yaml', ns: string, name: string): void }>()
</script>