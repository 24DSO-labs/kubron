<template>
  <div class="rd-row rd-click" @click="open = !open">
    <span class="rd-label">Job Spec<span class="rd-arrow" :class="{ open }">›</span></span>
    <span v-if="!open" class="rd-muted">{{ summary }}</span>
    <div v-else class="rd-wide" @click.stop>
      <div class="rd-row"><span class="rd-label">Parallelism</span><span class="rd-value">{{ spec.parallelism ?? 1 }}</span></div>
      <div class="rd-row"><span class="rd-label">Completions</span><span class="rd-value">{{ spec.completions ?? 1 }}</span></div>
      <div class="rd-row"><span class="rd-label">Backoff Limit</span><span class="rd-value">{{ spec.backoffLimit ?? 6 }}</span></div>
      <div class="rd-row"><span class="rd-label">Active</span><span class="rd-value">{{ status.active ?? 0 }}</span></div>
      <div class="rd-row"><span class="rd-label">Succeeded</span><span class="rd-value">{{ status.succeeded ?? 0 }}</span></div>
      <div class="rd-row"><span class="rd-label">Failed</span><span class="rd-value">{{ status.failed ?? 0 }}</span></div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

const props = defineProps<{ resource: any }>()
const open = ref(false)
const spec = computed(() => props.resource?.spec ?? {})
const status = computed(() => props.resource?.status ?? {})
const summary = computed(() => {
  const s = status.value
  return `active ${s.active ?? 0}, succeeded ${s.succeeded ?? 0}, failed ${s.failed ?? 0}`
})
</script>
