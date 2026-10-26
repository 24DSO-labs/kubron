<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="subsets.length" class="eps">
    <div v-for="(s, i) in subsets" :key="i" class="eps-subset">
      <div class="rd-row">
        <span class="rd-label">Ports</span>
        <span class="rd-value rd-mono">{{ ports(s) }}</span>
      </div>
      <div class="rd-table-wrap">
        <table class="rd-table">
          <thead><tr><th>IP</th><th>Hostname</th><th>Node</th><th>Target</th><th>State</th></tr></thead>
          <tbody>
            <tr v-for="(a, j) in s.addresses" :key="j">
              <td class="rd-mono">{{ a.ip }}</td>
              <td>{{ a.hostname || '—' }}</td>
              <td>
                <a v-if="a.node" class="rd-link" @click="emit('open', { id: 'node', ns: '', name: a.node })">{{ a.node }}</a>
                <template v-else>—</template>
              </td>
              <td>
                <a v-if="a.target" class="rd-link" @click="openRef(a.target)">{{ a.target.kind }}/{{ a.target.name }}</a>
                <template v-else>—</template>
              </td>
              <td><span class="rd-badge" :class="a.ready ? 'rd-ok' : 'rd-warn'">{{ a.ready ? 'Ready' : 'Not ready' }}</span></td>
            </tr>
          </tbody>
        </table>
      </div>
    </div>
  </div>
  <div v-else class="rd-row"><span class="rd-label">Subsets</span><span class="rd-muted">—</span></div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string }): void }>()

const subsets = computed<any[]>(() => props.resource?.subsets ?? [])
const ports = (s: any) =>
  (s.ports ?? []).map((p: any) => `${p.name ? p.name + ':' : ''}${p.port}/${p.protocol || 'TCP'}`).join(', ') || '—'
const openRef = (r: any) => emit('open', { id: String(r.kind).toLowerCase(), ns: r.namespace, name: r.name })
</script>

<style scoped>
.eps-subset { margin-bottom: 12px; }
</style>