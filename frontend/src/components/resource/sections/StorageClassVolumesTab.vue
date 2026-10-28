<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="rows.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr>
          <th class="sc-sort" @click="asc = !asc">
            {{ $t('common.name') }} <span class="sc-arrow">{{ asc ? '▲' : '▼' }}</span>
          </th>
          <th>{{ $t('defs.columns.capacity') }}</th>
          <th>{{ $t('common.status') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr
          v-for="v in rows"
          :key="v.name"
          class="rd-click"
          @click="emit('open', { id: 'persistentvolume', ns: '', name: v.name })"
        >
          <td>{{ v.name }}</td>
          <td>{{ v.capacity || '—' }}</td>
          <td :style="{ color: statusColor(v.status) }">{{ v.status || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ $t('sections.storageClassVolumes.empty') }}</div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { statusColor } from '../../../resources/status'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const asc = ref(true)

const rows = computed<{ name: string; capacity: string; status: string }[]>(() => {
  const list = [...(props.resource?.volumes ?? [])]
  list.sort((a, b) => a.name.localeCompare(b.name) * (asc.value ? 1 : -1))
  return list
})
</script>

<style scoped>
.sc-sort { cursor: pointer; user-select: none; }
.sc-arrow { margin-left: 4px; font-size: 0.65rem; }
</style>