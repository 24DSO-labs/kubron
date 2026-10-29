<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="items.length" class="rd-row">
    <span class="rd-label">{{ $t('defs.columns.conditions') }}</span>
    <div class="rd-badges">
      <span v-for="c in items" :key="c.label" class="rd-badge" :class="c.cls" :title="c.title">{{ c.label }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { t } from '../../../i18n'

const props = defineProps<{ resource: any }>()

/**
 * Для узла «нормально» — Ready=True при False у остальных (MemoryPressure, DiskPressure ...).
 * Поэтому показываем Ready и только те остальные condition, которые сейчас True (проблемы).
 */
const items = computed(() => {
  const conds: any[] = Array.isArray(props.resource?.conditions) ? props.resource.conditions : []
  const out: { label: string; cls: string; title: string }[] = []

  const ready = conds.find(c => c.type === 'Ready')
  if (ready) {
    const label = ready.status === 'True' ? 'Ready' : ready.status === 'Unknown' ? 'Unknown' : 'NotReady'
    out.push({
      label,
      cls: ready.status === 'True' ? 'rd-ok' : 'rd-err',
      title: ready.message || ready.reason || '',
    })
  }
  for (const c of conds) {
    if (c.type !== 'Ready' && c.status === 'True') {
      out.push({ label: c.type, cls: 'rd-err', title: c.message || c.reason || '' })
    }
  }
  if (props.resource?.unschedulable) {
    out.push({ label: 'SchedulingDisabled', cls: 'rd-warn', title: t('sections.nodeConditions.cordoned') })
  }
  return out
})
</script>
