<template>
  <div v-if="provider" class="pv-tab">
    <div class="rd-row pv-row">
      <span class="rd-label">Source Provider</span>
      <span class="rd-value">{{ provider.source }}</span>
    </div>
    <div v-for="f in provider.fields" :key="f.name" class="rd-row pv-row">
      <span class="rd-label" :title="f.name">{{ f.name }}</span>
      <span class="rd-value">{{ f.value }}</span>
    </div>
  </div>
  <div v-else class="rd-empty">No provider data</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const provider = computed<{ source: string; fields: { name: string; value: string }[] } | null>(
  () => props.resource?.provider ?? null,
)
</script>

<style scoped>
.pv-tab { padding: 4px 0; }
/* длинные ключи (storage.kubernetes.io/csiProvisionerIdentity) не обрезаем */
.pv-row { grid-template-columns: minmax(150px, 40%) 1fr; }
.pv-row .rd-label { white-space: normal; overflow: visible; word-break: break-all; }
</style>