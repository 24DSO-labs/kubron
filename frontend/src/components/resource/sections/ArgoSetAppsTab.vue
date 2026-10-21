<template>
  <div v-if="apps.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>Application</th><th>Sync</th><th>Health</th><th>Message</th></tr>
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
  <div v-else class="rd-empty">No applications generated</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { badgeClass } from '../../../resources/status'
import type { ArgoApplicationSetInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationSetInfo> }>()
const apps = computed(() => props.resource?.applications ?? [])
</script>
