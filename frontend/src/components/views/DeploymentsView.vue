<template>
  <div>
    <ContentHeader icon="🚀" title="Deployments" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Name</th><th>Namespace</th><th>Ready</th><th>Up-to-date</th><th>Strategy</th><th>Images</th><th>Age</th><th>Scale</th><th>Actions</th></tr></thead>
      <tbody>
        <tr v-for="dep in items" :key="dep.namespace+dep.name">
          <td class="resource-name">{{dep.name}}</td>
          <td class="ns-label">{{dep.namespace}}</td>
          <td><span :class="dep.readyReplicas===dep.replicas?'text-green':'text-yellow'">{{dep.readyReplicas}}/{{dep.replicas}}</span></td>
          <td>{{dep.updatedReplicas}}</td>
          <td>{{dep.strategy}}</td>
          <td class="images-cell"><div v-for="img in dep.images" :key="img" class="image-tag">{{img}}</div></td>
          <td>{{dep.age}}</td>
          <td class="scale-cell">
            <button class="scale-btn" @click="$emit('scale',dep,-1)">−</button>
            <span class="scale-count">{{dep.replicas}}</span>
            <button class="scale-btn" @click="$emit('scale',dep,1)">+</button>
          </td>
          <td class="actions-cell">
            <button class="btn-icon" @click="$emit('restart',dep.namespace,dep.name)" title="Restart">🔁</button>
            <button class="btn-icon" @click="$emit('revisions',dep.namespace,dep.name)" title="Revisions">📜</button>
            <button class="btn-icon" @click="$emit('yaml',dep.namespace,dep.name)" title="YAML">📋</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет Deployments</div>
  </div>
</template>

<script lang="ts" setup>
import type { DeploymentInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'

defineProps<{ items: DeploymentInfo[]; search: string }>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'scale', dep: DeploymentInfo, delta: number): void
  (e: 'yaml', ns: string, name: string): void
  (e: 'restart', ns: string, name: string): void
  (e: 'revisions', ns: string, name: string): void
}>()

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>