<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="rows.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead><tr><th>{{ $t('defs.columns.kind') }}</th><th>{{ $t('common.name') }}</th><th>{{ $t('common.namespace') }}</th></tr></thead>
      <tbody>
        <tr
          v-for="r in rows"
          :key="`${r.group}/${r.kind}/${r.namespace}/${r.name}`"
          :class="{ 'rd-click': !!target(r) }"
          @click="openRef(r)"
        >
          <td>{{ r.kind }}</td>
          <td :class="{ 'rd-link': !!target(r) }">{{ r.name }}</td>
          <td class="rd-muted">{{ r.namespace || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ $t('helm.resources.empty') }}</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { getResourceById } from '../../../resources/registry'

interface Ref { group: string; kind: string; namespace: string; name: string }

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string }): void }>()
const rows = computed<Ref[]>(() => props.resource?.resources ?? [])

/** Открываем только типы, которые приложение умеет показывать */
function target(r: Ref): { id: string; ns: string; name: string } | null {
  const def = getResourceById(r.kind)
  if (!def || !r.name) return null
  if (def.namespaced && !r.namespace) return null
  return { id: def.id, ns: def.namespaced ? r.namespace : '', name: r.name }
}
function openRef(r: Ref) {
  const t = target(r)
  if (t) emit('open', t)
}
</script>