<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="lr">
    <div class="lr-title">Limits</div>
    <div v-if="!limits.length" class="rd-muted">No limits defined</div>
    <div v-else class="rd-table-wrap">
      <table class="rd-table">
        <thead>
          <tr>
            <th>Type</th><th>Resource</th><th>Min</th><th>Max</th>
            <th>Default</th><th>Default request</th><th>Max limit/request ratio</th>
          </tr>
        </thead>
        <tbody>
          <tr v-for="(l, i) in limits" :key="i">
            <td>{{ l.type }}</td>
            <td class="rd-mono">{{ l.resource }}</td>
            <td>{{ dash(l.min) }}</td>
            <td>{{ dash(l.max) }}</td>
            <td>{{ dash(l.default) }}</td>
            <td>{{ dash(l.defaultRequest) }}</td>
            <td>{{ dash(l.maxLimitRequestRatio) }}</td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const limits = computed<any[]>(() => (Array.isArray(props.resource?.limits) ? props.resource.limits : []))
const dash = (v?: string) => (v ? v : '—')
</script>

<style scoped>
.lr-title { font-weight: 700; color: #fff; padding: 12px 0 8px; text-align: left; }
</style>
