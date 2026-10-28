<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="rd-row">
    <span class="rd-label">{{ $t('sections.images.title') }}</span>
    <div class="rd-value rd-pairs">
      <div v-for="img in images" :key="img" class="rd-mono">{{ img }}</div>
      <span v-if="!images.length" class="rd-muted">—</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useControlledPods } from '../../../resources/useControlledPods'

const props = defineProps<{ resource: any }>()

/** Образы из самого ресурса; если их там нет, собираем из контейнеров его подов */
const own = computed<string[]>(() => {
  const v = props.resource?.images
  if (Array.isArray(v)) return v.filter(Boolean)
  return typeof v === 'string' && v ? [v] : []
})

const { pods } = useControlledPods(() => (own.value.length ? null : props.resource))

const images = computed<string[]>(() => {
  if (own.value.length) return own.value
  const set = new Set<string>()
  for (const p of pods.value) for (const c of p.containers ?? []) if (c.image) set.add(c.image)
  return [...set].sort()
})
</script>
