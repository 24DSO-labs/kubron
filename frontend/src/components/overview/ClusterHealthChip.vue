<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <span class="chip" :class="tone" :title="detail" role="status">
    <span class="dot"></span>
    <template v-if="bad.length">{{ $t('overview.health.chipBad', { names: bad.map(c => c.name).join(', ') }) }}</template>
    <template v-else>{{ $t('overview.health.chipOk') }}</template>
  </span>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClusterHealth } from '../../types/overview'

const props = defineProps<{ health: ClusterHealth }>()
const { t, te } = useI18n()
const bad = computed(() => props.health.components.filter(c => c.status === 'unhealthy'))
const tone = computed(() => (bad.value.length ? 'bad' : 'ok'))
const detail = computed(() => props.health.components
    .map(c => `${c.name}: ${te(`overview.health.component.${c.status}`) ? t(`overview.health.component.${c.status}`) : c.status}`)
    .join('\n'))
</script>

<style scoped>
.chip { display: inline-flex; align-items: center; gap: 6px; padding: 3px 10px; border-radius: 999px; font-size: 12px; font-weight: 500; }
.dot { width: 6px; height: 6px; border-radius: 50%; background: currentColor; }
.ok { color: var(--ov-ok); background: color-mix(in srgb, var(--ov-ok) 12%, transparent); }
.bad { color: var(--ov-danger); background: color-mix(in srgb, var(--ov-danger) 14%, transparent); }
</style>
