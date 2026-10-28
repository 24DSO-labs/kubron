<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="js">
    <div v-for="r in rows" :key="r.label" class="rd-row">
      <span class="rd-label">{{ r.label }}</span>
      <span class="rd-value">{{ r.value === undefined || r.value === null || r.value === '' ? '—' : r.value }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { t } from '../../../i18n'
import { podStatusSummary } from '../../../resources/utils'

const props = defineProps<{ resource: any }>()

const rows = computed(() => [
  { label: t('defs.columns.completions'), value: props.resource?.completions },
  { label: t('sections.jobStats.parallelism'), value: props.resource?.parallelism },
  { label: t('sections.jobStats.backoffLimit'), value: props.resource?.backoffLimit },
  { label: t('defs.props.podStatus'), value: podStatusSummary(props.resource) },
])
</script>

<style scoped>
.js { display: contents; }
</style>