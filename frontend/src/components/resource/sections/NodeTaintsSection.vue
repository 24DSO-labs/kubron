<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="taints.length" class="rd-row">
    <span class="rd-label">{{ $t('defs.columns.taints') }}</span>
    <div class="rd-badges">
      <span v-for="(t, i) in taints" :key="i" class="rd-badge rd-unk" :title="t.effect">
        {{ taintText(t) }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()

const taints = computed<any[]>(() =>
  Array.isArray(props.resource?.taints) ? props.resource.taints : [],
)

function taintText(t: any): string {
  const kv = t.value ? `${t.key}=${t.value}` : t.key
  return `${kv}:${t.effect}`
}
</script>