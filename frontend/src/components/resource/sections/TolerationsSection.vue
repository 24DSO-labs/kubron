<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="rd-row rd-click" @click="open = !open">
    <span class="rd-label">{{ $t('sections.tolerations.title') }}<span class="rd-arrow" :class="{ open }">›</span></span>
    <span v-if="!open" class="rd-muted">{{ tolerations.length }}</span>
    <div v-else-if="tolerations.length" class="rd-table-wrap" @click.stop>
      <table class="rd-table">
        <thead>
          <tr><th>{{ $t('sections.tolerations.key') }}</th><th>{{ $t('sections.tolerations.operator') }}</th><th>{{ $t('defs.columns.value') }}</th><th>{{ $t('sections.tolerations.effect') }}</th><th>{{ $t('sections.tolerations.seconds') }}</th></tr>
        </thead>
        <tbody>
          <tr v-for="(t, i) in tolerations" :key="i">
            <td>{{ t.key || '' }}</td>
            <td>{{ t.operator || '' }}</td>
            <td>{{ t.value || '' }}</td>
            <td>{{ t.effect || '' }}</td>
            <td>{{ t.tolerationSeconds ?? '' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <span v-else class="rd-muted">—</span>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

const props = defineProps<{ resource: any }>()
const open = ref(false)
const tolerations = computed<any[]>(() =>
  Array.isArray(props.resource?.tolerations) ? props.resource.tolerations : [],
)
</script>
