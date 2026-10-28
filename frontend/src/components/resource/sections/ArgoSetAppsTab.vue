<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="apps.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>{{ $t('defs.columns.application') }}</th><th>{{ $t('argo.columns.sync') }}</th><th>{{ $t('argo.columns.health') }}</th><th>{{ $t('defs.columns.message') }}</th></tr>
      </thead>
      <tbody>
        <tr v-for="a in apps" :key="a.name">
          <td class="rd-mono">{{ a.name }}</td>
          <td><span v-if="a.status" class="rd-badge" :class="badgeClass(a.status)">{{ a.status }}</span><span v-else>—</span></td>
          <td><span v-if="a.health" class="rd-badge" :class="badgeClass(a.health)">{{ a.health }}</span><span v-else>—</span></td>
          <td class="rd-muted">{{ a.message || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ $t('argo.setApps.empty') }}</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { badgeClass } from '../../../resources/status'
import type { ArgoApplicationSetInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationSetInfo> }>()
const apps = computed(() => props.resource?.applications ?? [])
</script>
