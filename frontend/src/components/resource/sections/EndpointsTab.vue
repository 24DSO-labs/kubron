<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="rows.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead><tr><th>{{ $t('common.name') }}</th><th>{{ $t('defs.columns.endpoints') }}</th></tr></thead>
      <tbody>
        <tr
          v-for="r in rows"
          :key="r.name"
          class="rd-click"
          @click="emit('open', { id: 'endpoint', ns: resource.namespace, name: r.name })"
        >
          <td>{{ r.name }}</td>
          <td class="rd-mono">{{ r.addresses.join(', ') || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ $t('sections.endpointsTab.empty') }}</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

// Go отдаёт по записи на subset с одним и тем же именем: сводим в одну строку
const rows = computed(() => {
  const byName = new Map<string, string[]>()
  for (const ep of (props.resource?.endpoints ?? []) as any[]) {
    byName.set(ep.name, [...(byName.get(ep.name) ?? []), ...(ep.addresses ?? [])])
  }
  return [...byName].map(([name, addresses]) => ({ name, addresses }))
})
</script>