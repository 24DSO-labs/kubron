<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>📜 Revisions</h3>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>
    <table v-if="revisions.length>0" class="table">
      <thead><tr><th>Revision</th><th>Name</th><th>Replicas</th><th>Ready</th><th>Images</th><th>Age</th><th></th></tr></thead>
      <tbody>
        <tr v-for="rev in revisions" :key="rev.revision">
          <td><span class="badge">{{rev.revision}}</span></td>
          <td class="resource-name">{{rev.name}}</td>
          <td>{{rev.replicas}}</td>
          <td>{{rev.ready}}</td>
          <td class="images-cell"><div class="image-tag">{{rev.images}}</div></td>
          <td>{{rev.age}}</td>
          <td><button class="btn btn-sm" @click="$emit('rollback', Number(rev.revision))">↩ Rollback</button></td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет ревизий</div>
  </div>
</template>

<script lang="ts" setup>
import type { RevisionInfo } from '../../../types'
defineProps<{ revisions: RevisionInfo[] }>()
defineEmits<{ (e: 'close'): void; (e: 'rollback', rev: number): void }>()
</script>