<template>
  <div class="pf-section">
    <h4>🌍 Port Forward</h4>
    <div class="pf-form">
      <label>Local:</label>
      <input type="number" :value="localPort" @input="$emit('update:localPort', Number(($event.target as HTMLInputElement).value))" class="pf-input"/>
      <label>Pod:</label>
      <input type="number" :value="podPort" @input="$emit('update:podPort', Number(($event.target as HTMLInputElement).value))" class="pf-input"/>
      <button @click="$emit('start')" class="btn btn-sm">▶ Start</button>
    </div>
    <div v-if="sessions.length>0" class="pf-list">
      <div v-for="s in sessions" :key="s.id" class="pf-item">
        <span class="pf-info">localhost:{{s.localPort}} → {{s.pod}}:{{s.podPort}}</span>
        <span class="status-badge" :class="s.status==='active'?'status-running':'status-pending'">{{s.status}}</span>
        <button class="btn-icon" @click="$emit('stop', s.id)">⏹</button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { PortForwardSession } from '../../types'
defineProps<{
  localPort: number
  podPort: number
  sessions: PortForwardSession[]
}>()
defineEmits<{
  (e: 'start'): void
  (e: 'stop', id: string): void
  (e: 'update:localPort', val: number): void
  (e: 'update:podPort', val: number): void
}>()
</script>