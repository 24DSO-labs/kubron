<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="sources.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>#</th><th>Repository</th><th>Path / Chart</th><th>Target revision</th><th>Ref</th></tr>
      </thead>
      <tbody>
        <tr v-for="(s, i) in sources" :key="i">
          <td class="rd-muted">{{ i + 1 }}</td>
          <td class="rd-mono">{{ shortRepo(s.repoURL) || '—' }}</td>
          <td class="rd-mono">{{ argoPathOrChart(s) || '—' }}</td>
          <td class="rd-mono">{{ s.targetRevision || 'HEAD' }}</td>
          <td class="rd-mono">{{ s.ref || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">No sources</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { argoPathOrChart, shortRepo } from '../../../resources/argocd'
import type { ArgoApplicationInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()
const sources = computed(() => props.resource?.sources ?? [])
</script>
