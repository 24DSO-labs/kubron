<template>
  <div class="connect-screen">
    <h2>Подключение к кластеру</h2>
    <button @click="$emit('load')" class="btn">Загрузить контексты</button>
    <div v-if="contexts.length > 0" class="contexts">
      <div v-for="ctx in contexts" :key="ctx.name" @click="$emit('connect', ctx.name)" class="context-item">
        <span class="name">{{ ctx.name }}</span>
        <span class="cluster">{{ ctx.cluster }}</span>
        <span v-if="ctx.isCurrent" class="badge">current</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ContextInfo } from '../types'
defineProps<{ contexts: ContextInfo[] }>()
defineEmits<{ (e: 'load'): void; (e: 'connect', name: string): void }>()
</script>