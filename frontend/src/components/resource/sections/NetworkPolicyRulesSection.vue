<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="np">
    <template v-for="d in dirs" :key="d.key">
      <div class="np-title">{{ d.title }}</div>
      <div v-if="!d.active" class="rd-muted">Not applied by this policy</div>
      <div v-else-if="!d.rules.length" class="rd-muted">No rules: all {{ d.key }} traffic is denied</div>
      <div v-else class="rd-table-wrap">
        <table class="rd-table">
          <thead><tr><th>{{ d.peerLabel }}</th><th>Ports</th></tr></thead>
          <tbody>
            <tr v-for="(r, i) in d.rules" :key="i">
              <td>
                <div v-for="p in r.peers" :key="p" class="rd-mono">{{ p }}</div>
                <span v-if="!r.peers?.length" class="rd-muted">All</span>
              </td>
              <td>
                <div v-for="p in r.ports" :key="p" class="rd-mono">{{ p }}</div>
                <span v-if="!r.ports?.length" class="rd-muted">All</span>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const types = computed<string[]>(() => props.resource?.policyTypes ?? [])

const dirs = computed(() => [
  { key: 'ingress', title: 'Ingress rules', peerLabel: 'From', active: types.value.includes('Ingress'), rules: props.resource?.ingress ?? [] },
  { key: 'egress', title: 'Egress rules', peerLabel: 'To', active: types.value.includes('Egress'), rules: props.resource?.egress ?? [] },
])
</script>

<style scoped>
.np-title { font-weight: 700; color: #fff; padding: 12px 0 8px; text-align: left; }
</style>