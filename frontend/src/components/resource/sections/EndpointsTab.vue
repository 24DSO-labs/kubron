<template>
  <div class="ep-tab">
    <div v-if="!endpoints.length" class="ep-empty">No Endpoints</div>
    <div v-for="ep in endpoints" :key="ep.name" class="ep-card">
      <div class="ep-name">{{ ep.name }}</div>
      <div v-if="ep.addresses?.length" class="ep-addrs">
        <span v-for="addr in ep.addresses" :key="addr" class="ep-addr">{{ addr }}</span>
      </div>
      <div v-else class="ep-noaddrs">—</div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const endpoints = computed(() => props.resource?.endpoints ?? [])
</script>

<style scoped>
.ep-tab { padding: 12px 16px; }
.ep-empty { color: #868d8a; font-size: 0.875rem; padding: 16px 0; }
.ep-card {
  background: #1a1b2e;
  border: 1px solid #2a2b40;
  border-radius: 8px;
  padding: 14px 16px;
  margin-bottom: 10px;
}
.ep-name {
  font-weight: 600;
  color: #d4d4d4;
  font-size: 0.9rem;
  margin-bottom: 8px;
}
.ep-addrs {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.ep-addr {
  display: inline-block;
  padding: 3px 10px;
  background: #252640;
  border-radius: 4px;
  font-size: 0.82rem;
  color: #c0c2d8;
  font-family: 'Courier New', monospace;
}
.ep-noaddrs { color: #868d8a; font-size: 0.82rem; }
</style>
