<template>
  <span class="chip" :class="tone" :title="detail" role="status">
    <span class="dot"></span>
    <template v-if="bad.length">{{ bad.map(c => c.name).join(', ') }}: unhealthy</template>
    <template v-else>Control plane healthy</template>
  </span>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ClusterHealth } from '../../types/overview'

const props = defineProps<{ health: ClusterHealth }>()
const bad = computed(() => props.health.components.filter(c => c.status === 'unhealthy'))
const tone = computed(() => (bad.value.length ? 'bad' : 'ok'))
const detail = computed(() => props.health.components.map(c => `${c.name}: ${c.status}`).join('\n'))
</script>

<style scoped>
.chip { display: inline-flex; align-items: center; gap: 6px; padding: 3px 10px; border-radius: 999px; font-size: 12px; font-weight: 500; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.ok { color: var(--ov-ok); background: color-mix(in srgb, var(--ov-ok) 12%, transparent); }
.bad { color: var(--ov-danger); background: color-mix(in srgb, var(--ov-danger) 14%, transparent); }
</style>
