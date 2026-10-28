<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="entries.length" class="sc-tab">
    <div v-for="[k, v] in entries" :key="k" class="rd-row sc-row">
      <span class="rd-label" :title="k">{{ k }}</span>
      <span class="rd-value">{{ v === '' ? '—' : v }}</span>
    </div>
  </div>
  <div v-else class="rd-empty">{{ $t('sections.storageClassParameters.empty') }}</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()

const entries = computed(() =>
  Object.entries((props.resource?.parameters ?? {}) as Record<string, string>).sort(([a], [b]) =>
    a.localeCompare(b),
  ),
)
</script>

<style scoped>
.sc-tab { padding: 4px 0; }
/* длинные ключи (csi.storage.k8s.io/...) переносим, а не обрезаем */
.sc-row { grid-template-columns: minmax(150px, 40%) 1fr; }
.sc-row .rd-label { white-space: normal; overflow: visible; word-break: break-all; }
</style>