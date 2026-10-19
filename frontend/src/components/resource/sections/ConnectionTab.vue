<template>
  <div class="conn-tab">
    <div class="rd-row">
      <span class="rd-label">Cluster IP</span>
      <span class="rd-value">{{ resource.clusterIP || '—' }}</span>
    </div>
    <div v-if="resource.clusterIPs?.length" class="rd-row">
      <span class="rd-label">Cluster IPs</span>
      <span class="rd-value">{{ resource.clusterIPs.join(', ') }}</span>
    </div>
    <div class="rd-row">
      <span class="rd-label">IP families</span>
      <span class="rd-value">{{ resource.ipFamilies || '—' }}</span>
    </div>
    <div class="rd-row">
      <span class="rd-label">IP family policy</span>
      <span class="rd-value">{{ resource.ipFamilyPolicy || '—' }}</span>
    </div>
    <div class="rd-row">
      <span class="rd-label">External IPs</span>
      <span class="rd-value">{{ resource.externalIPs?.join(', ') || '—' }}</span>
    </div>
    <div v-if="ports.length" class="rd-row">
      <span class="rd-label">Ports</span>
      <div class="rd-value conn-ports">
        <div v-for="p in ports" :key="p.port + '/' + p.protocol" class="conn-port">
          <span class="conn-port-main">{{ formatPort(p) }}</span>
          <PortForwardControl
            :namespace="resource.namespace"
            :resource-name="resource.name"
            :port="p.port"
            resource-type="service"
          />
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import PortForwardControl from './PortForwardControl.vue'

const props = defineProps<{ resource: any }>()

const ports = computed(() => props.resource?.ports ?? [])

function formatPort(p: any): string {
  let s = `${p.port}`
  if (p.nodePort) {
    s += `:${p.nodePort}`
  }
  s += `/${p.protocol || 'TCP'}`
  return s
}
</script>

<style scoped>
.conn-tab { padding: 12px 16px; }
.conn-ports { display: flex; flex-direction: column; gap: 6px; }
.conn-port { display: flex; align-items: center; gap: 12px; flex-wrap: wrap; }
.conn-port-main { color: #60a5fa; font-weight: 500; font-size: 0.875rem; }
</style>
