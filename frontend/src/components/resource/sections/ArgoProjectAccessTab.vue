<template>
  <div class="rd-stack">
    <h4 class="rd-subtitle">Source repositories</h4>
    <div v-if="p.sourceRepos?.length" class="rd-badges">
      <span v-for="r in p.sourceRepos" :key="r" class="rd-badge rd-unk rd-mono">{{ r }}</span>
    </div>
    <div v-else class="rd-empty">None allowed</div>

    <h4 class="rd-subtitle">Destinations</h4>
    <div v-if="p.destinations?.length" class="rd-table-wrap">
      <table class="rd-table">
        <thead><tr><th>Cluster</th><th>Namespace</th></tr></thead>
        <tbody>
          <tr v-for="(d, i) in p.destinations" :key="i">
            <td class="rd-mono">{{ d.name || d.server || '*' }}</td>
            <td class="rd-mono">{{ d.namespace || '*' }}</td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="rd-empty">None allowed</div>

    <template v-for="l in lists" :key="l.title">
      <template v-if="l.items?.length">
        <h4 class="rd-subtitle">{{ l.title }}</h4>
        <div class="rd-badges">
          <span v-for="(g, i) in l.items" :key="i" class="rd-badge rd-unk rd-mono">{{ g.group || 'core' }}/{{ g.kind }}</span>
        </div>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ArgoProjectInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoProjectInfo> }>()
const p = computed(() => props.resource ?? {})
const lists = computed(() => [
  { title: 'Cluster resource whitelist', items: p.value.clusterResourceWhitelist },
  { title: 'Cluster resource blacklist', items: p.value.clusterResourceBlacklist },
  { title: 'Namespace resource whitelist', items: p.value.namespaceResourceWhitelist },
  { title: 'Namespace resource blacklist', items: p.value.namespaceResourceBlacklist },
])
</script>
