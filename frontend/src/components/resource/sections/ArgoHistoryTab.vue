<template>
  <div v-if="history.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>ID</th><th>Revision</th><th>Deployed</th><th>Source</th></tr>
      </thead>
      <tbody>
        <tr v-for="h in history" :key="h.id">
          <td class="rd-mono">#{{ h.id }}</td>
          <td class="rd-mono" :title="h.revision">{{ shortRevision(h.revision) || '—' }}</td>
          <td class="rd-muted">{{ h.deployedAt || '—' }}</td>
          <td class="rd-mono">{{ shortRepoSource(h.source) }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">No deployment history</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { shortRevision } from '../../../resources/argocd'
import type { ArgoApplicationInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()
// Бэкенд уже отдаёт новые записи первыми
const history = computed(() => props.resource?.history ?? [])

/** "https://host/org/repo@main (path)" -> без схемы; остальное как есть */
const shortRepoSource = (s: string) => (s || '—').replace(/^[a-z+]+:\/\//i, '')
</script>
