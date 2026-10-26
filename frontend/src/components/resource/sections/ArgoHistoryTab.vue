<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="history.length">
    <div v-if="resource.autoSync" class="rd-muted ah-hint">Rollback is unavailable while auto-sync is enabled.</div>
    <div class="rd-table-wrap">
      <table class="rd-table">
        <thead>
          <tr><th>ID</th><th>Revision</th><th>Deployed</th><th>Source</th><th></th></tr>
        </thead>
        <tbody>
          <tr v-for="(h, i) in history" :key="h.id">
            <td class="rd-mono">#{{ h.id }}</td>
            <td class="rd-mono" :title="h.revision">{{ shortRevision(h.revision) || '—' }}</td>
            <td class="rd-muted">{{ h.deployedAt || '—' }}</td>
            <td class="rd-mono">{{ shortRepoSource(h.source) }}</td>
            <td>
              <!-- самая новая запись — текущая ревизия, откатываться на неё бессмысленно -->
              <button v-if="canRollback && i > 0" class="rd-btn" @click="onRollback(h)">Rollback</button>
            </td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
  <div v-else class="rd-empty">No deployment history</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { shortRevision } from '../../../resources/argocd'
import { rollbackArgo } from '../../../resources/argoActions'
import { can } from '../../../composables/useAuth'
import { Perm } from '../../../types/auth'
import type { ArgoApplicationInfo, ArgoHistoryEntry } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()
// Бэкенд уже отдаёт новые записи первыми
const history = computed(() => props.resource?.history ?? [])
const canRollback = computed(() => can(Perm.Operate) && !props.resource?.autoSync && !props.resource?.operationOn)

/** "https://host/org/repo@main (path)" -> без схемы; остальное как есть */
const shortRepoSource = (s: string) => (s || '—').replace(/^[a-z+]+:\/\//i, '')

const onRollback = (h: ArgoHistoryEntry) =>
  rollbackArgo({ namespace: props.resource?.namespace ?? '', name: props.resource?.name ?? '' }, h)
</script>

<style scoped>
.ah-hint { padding: 8px 0; font-size: 0.8rem; text-align: left; }
</style>