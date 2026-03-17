<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>🌐 {{ svc.name }}</h3>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

       <!-- Connection -->
    <div class="detail-section">
      <h4>Connection</h4>
      <table class="detail-table">
        <tbody>
          <tr><td class="dt-label">Cluster IP</td><td>{{ svc.clusterIP }}</td></tr>
          <tr><td class="dt-label">Cluster IPs</td><td>{{ svc.clusterIPs?.join(', ') }}</td></tr>
          <tr><td class="dt-label">IP families</td><td>{{ svc.ipFamilies }}</td></tr>
          <tr><td class="dt-label">IP family policy</td><td>{{ svc.ipFamilyPolicy }}</td></tr>
          <tr><td class="dt-label">Type</td><td><span class="type-badge">{{ svc.type }}</span></td></tr>
          <tr><td class="dt-label">Age</td><td>{{ svc.age }}</td></tr>
        </tbody>
      </table>
    </div>

    <!-- Ports -->
    <div class="detail-section">
      <h4>Ports</h4>
      <div v-for="p in svc.ports" :key="p.port" class="port-row">
        <div class="port-info">
          <span class="port-label">{{ p.port }}:{{ p.targetPort }}/{{ p.protocol }}</span>
          <span v-if="p.name" class="port-name">({{ p.name }})</span>
          <span v-if="p.nodePort" class="port-nodeport">NodePort: {{ p.nodePort }}</span>
        </div>
        <div class="port-actions">
          <div v-if="isForwarded(p.port)" class="pf-active">
            <span class="pf-active-label">→ localhost:{{ getForwardedPort(p.port) }}</span>
            <button class="btn btn-sm btn-danger" @click="$emit('stopForward', getForwardId(p.port))">Stop</button>
          </div>
          <button v-else class="btn btn-sm" @click="$emit('forward', svc.namespace, svc.name, p.port)">
            🌍 Forward...
          </button>
        </div>
      </div>
    </div>

    <!-- Selector -->
    <div class="detail-section" v-if="svc.selector && Object.keys(svc.selector).length">
      <h4>Selector</h4>
      <div class="label-list">
        <span v-for="(v,k) in svc.selector" :key="k" class="label-tag">{{ k }}={{ v }}</span>
      </div>
    </div>

    <!-- Endpoints -->
    <div class="detail-section">
      <h4>Endpoints</h4>
      <div v-if="svc.endpoints?.length">
        <div v-for="ep in svc.endpoints" :key="ep.name" class="endpoint-item">
          <span class="endpoint-name">{{ ep.name }}</span>
          <span class="endpoint-addrs">{{ ep.addresses?.join(', ') }}</span>
        </div>
      </div>
      <div v-else class="empty-small">No endpoints found</div>
    </div>

    <!-- Labels -->
    <div class="detail-section" v-if="svc.labels && Object.keys(svc.labels).length">
      <h4>Labels</h4>
      <div class="label-list">
        <span v-for="(v,k) in svc.labels" :key="k" class="label-tag">{{ k }}={{ v }}</span>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ServiceDetail, PortForwardSession } from '../../types'

const props = defineProps<{
  svc: ServiceDetail
  portForwards: PortForwardSession[]
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'forward', ns: string, name: string, port: number): void
  (e: 'stopForward', id: string): void
}>()

function isForwarded(port: number): boolean {
  return props.portForwards.some(pf =>
    pf.pod.includes(props.svc.name) && pf.podPort === port && pf.status === 'active'
  )
}

function getForwardedPort(port: number): number {
  const pf = props.portForwards.find(pf =>
    pf.pod.includes(props.svc.name) && pf.podPort === port
  )
  return pf?.localPort || port
}

function getForwardId(port: number): string {
  const pf = props.portForwards.find(pf =>
    pf.pod.includes(props.svc.name) && pf.podPort === port
  )
  return pf?.id || ''
}
</script>