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
import type { ArgoApplicationInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()
const conditions = computed(() => props.resource?.conditions ?? [])

/**
 * У Application condition нет status: присутствие уже означает проблему.
 * Типы *Error — ошибка (красный), остальные (*Warning, ExcludedResourceWarning, OrphanedResourceWarning) — предупреждение.
 */
function toneOf(type: string): string {
  return /error$/i.test(type) ? 'rd-err' : 'rd-warn'
}
</script>
