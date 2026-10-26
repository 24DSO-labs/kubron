<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="rules.length" class="rules">
    <div class="rules-title">Rules</div>
    <table class="rules-table">
      <thead>
        <tr><th>API groups</th><th>Resources</th><th>Verbs</th></tr>
      </thead>
      <tbody>
        <tr v-for="(r, i) in rules" :key="i">
          <td>{{ join(r.apiGroups) }}</td>
          <td>{{ join(r.resources ?? r.nonResourceURLs) }}</td>
          <td>{{ join(r.verbs) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()

const rules = computed<any[]>(() =>
  Array.isArray(props.resource?.rules) ? props.resource.rules : [],
)

function join(v: unknown, empty = '—'): string {
  if (!Array.isArray(v) || v.length === 0) return empty
  return v.map(x => (x === '' ? '(core)' : x)).join(', ')
}
</script>

<style scoped>
.rules-title { font-weight: bold; color: #d4d4d4; margin-bottom: 8px; font-size: 0.875rem; }
.rules-table { width: 100%; border-collapse: collapse; font-size: 0.8125rem; }
.rules-table th, .rules-table td {
  text-align: left; vertical-align: top;
  padding: 6px 8px; border-bottom: 1px solid #3b3d5c;
}
.rules-table th { color: #868d8a; font-weight: 500; }
</style>
