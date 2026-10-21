<template>
  <div v-if="conditions.length" class="rd-row">
    <span class="rd-label">Conditions</span>
    <div class="rd-badges">
      <span
        v-for="c in conditions"
        :key="c.type"
        class="rd-badge"
        :class="toneOf(c.type)"
        :title="c.message || ''"
      >{{ c.type }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ArgoApplicationInfo, ArgoApplicationSetInfo } from '../../../types/argocd'

// Application (condition = «активно», status всегда True) и ApplicationSet (у condition есть status True/False)
const props = defineProps<{ resource: Partial<ArgoApplicationInfo | ArgoApplicationSetInfo> }>()
const conditions = computed(() => (props.resource?.conditions ?? []).filter(c => c.status !== 'False'))

/**
 * Application: присутствие condition уже означает проблему (*Error — красный, остальные — предупреждение).
 * ApplicationSet: ErrorOccurred — ошибка, штатные ResourcesUpToDate/ParametersGenerated/RolloutProgressing — норма.
 */
const NORMAL = ['ResourcesUpToDate', 'ParametersGenerated', 'RolloutProgressing']
function toneOf(type: string): string {
  if (/error/i.test(type)) return 'rd-err'
  return NORMAL.includes(type) ? 'rd-ok' : 'rd-warn'
}
</script>
