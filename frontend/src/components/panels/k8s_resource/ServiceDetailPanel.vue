<template>
  <K8sResourceDetailBase
    :resource="svc"
    resource-icon="🌐"
    @close="$emit('close')"
  >
    <template #actions>
      <button class="btn-action" title="Edit YAML" @click="$emit('edit')">✏️</button>
      <button class="btn-action btn-danger" title="Delete" @click="$emit('delete')">🗑️</button>
    </template>

    <template #properties>
      <div class="svc-properties">
        <!-- Connection section -->
        <div class="section">
          <h4>Connection</h4>
          <table class="detail-table">
            <tbody>
              <tr>
                <td class="dt-label">Cluster IP</td>
                <td>{{ svc.clusterIP }}</td>
              </tr>
              <tr>
                <td class="dt-label">Cluster IPs</td>
                <td>{{ svc.clusterIPs?.join(', ') }}</td>
              </tr>
              <tr>
                <td class="dt-label">IP families</td>
                <td>{{ svc.ipFamilies }}</td>
              </tr>
              <tr>
                <td class="dt-label">IP family policy</td>
                <td>{{ svc.ipFamilyPolicy }}</td>
              </tr>
              <tr>
                <td class="dt-label">Type</td>
                <td><span class="type-badge">{{ svc.type }}</span></td>
              </tr>
            </tbody>
          </table>
        </div>

        <!-- Ports section -->
        <div class="section">
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
                <button class="btn btn-sm btn-danger" @click="$emit('stopForward', getForwardId(p.port))">
                  Stop
                </button>
              </div>
              <button v-else class="btn btn-sm" @click="$emit('forward', svc.namespace, svc.name, p.port)">
                🌍 Forward...
              </button>
            </div>
          </div>
        </div>

        <!-- Selector section -->
        <div v-if="svc.selector && Object.keys(svc.selector).length" class="section">
          <h4>Selector</h4>
          <div class="label-list">
            <span v-for="(v, k) in svc.selector" :key="k" class="label-tag">{{ k }}={{ v }}</span>
          </div>
        </div>

        <!-- Endpoints section -->
        <div class="section">
          <h4>Endpoints</h4>
          <div v-if="svc.endpoints?.length">
            <div v-for="ep in svc.endpoints" :key="ep.name" class="endpoint-item">
              <span class="endpoint-name">{{ ep.name }}</span>
              <span class="endpoint-addrs">{{ ep.addresses?.join(', ') }}</span>
            </div>
          </div>
          <div v-else class="empty-small">No endpoints found</div>
        </div>

        <!-- Labels section -->
        <div v-if="svc.labels && Object.keys(svc.labels).length" class="section">
          <h4>Labels</h4>
          <div class="label-list">
            <span v-for="(v, k) in svc.labels" :key="k" class="label-tag">{{ k }}={{ v }}</span>
          </div>
        </div>
      </div>
    </template>

    <template #events>
      <div class="empty-section">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import type { ServiceDetail, PortForwardSession } from '../../../types'

const props = defineProps<{
  svc: ServiceDetail
  portForwards: PortForwardSession[]
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'forward', ns: string, name: string, port: number): void
  (e: 'stopForward', id: string): void
}>()

function isForwarded(port: number): boolean {
  return props.portForwards.some(
    (pf) => pf.pod.includes(props.svc.name) && pf.podPort === port && pf.status === 'active'
  )
}

function getForwardedPort(port: number): number {
  const pf = props.portForwards.find((pf) => pf.pod.includes(props.svc.name) && pf.podPort === port)
  return pf?.localPort || port
}

function getForwardId(port: number): string {
  const pf = props.portForwards.find((pf) => pf.pod.includes(props.svc.name) && pf.podPort === port)
  return pf?.id || ''
}
</script>

<style scoped>
.svc-properties {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.section {
  margin-bottom: 8px;
}

.section h4 {
  margin: 0 0 12px 0;
  color: #e0e2f0;
  font-size: 0.95rem;
  font-weight: 600;
}

.detail-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.875rem;
}

.detail-table tbody tr {
  border-bottom: 1px solid #2a2b40;
}

.detail-table tbody tr:last-child {
  border-bottom: none;
}

.dt-label {
  font-weight: 600;
  color: #9ca0b8;
  width: 140px;
  padding: 6px 0;
  white-space: nowrap;
}

.detail-table td {
  padding: 6px 12px;
  color: #d4d4d4;
  text-align: left;
}

.detail-table td:first-child {
  padding-left: 0;
}

.type-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-radius: 3px;
  font-size: 0.8rem;
  font-weight: 600;
}

.port-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 0;
  border-bottom: 1px solid #2a2b40;
  gap: 12px;
}

.port-row:last-child {
  border-bottom: none;
}

.port-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
  flex-wrap: wrap;
}

.port-label {
  color: #60a5fa;
  font-weight: 500;
  font-size: 0.875rem;
}

.port-name {
  color: #9ca0b8;
  font-size: 0.8rem;
}

.port-nodeport {
  color: #f59e0b;
  font-size: 0.8rem;
  font-weight: 500;
}

.port-actions {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-shrink: 0;
}

.pf-active {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 8px;
  background: rgba(52, 211, 153, 0.1);
  border-radius: 3px;
}

.pf-active-label {
  color: #34d399;
  font-size: 0.8rem;
  font-weight: 500;
}

.btn {
  padding: 4px 8px;
  background: #2563eb;
  color: white;
  border: none;
  border-radius: 3px;
  font-size: 0.8rem;
  cursor: pointer;
  transition: background 0.15s;
}

.btn:hover {
  background: #1d4ed8;
}

.btn-sm {
  padding: 3px 6px;
  font-size: 0.75rem;
}

.btn-danger {
  background: #ef4444;
}

.btn-danger:hover {
  background: #dc2626;
}

.label-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.label-tag {
  display: inline-block;
  padding: 4px 8px;
  background: rgba(96, 165, 250, 0.15);
  color: #60a5fa;
  border-radius: 3px;
  font-size: 0.8rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.endpoint-item {
  padding: 6px 0;
  border-bottom: 1px solid #2a2b40;
  font-size: 0.875rem;
}

.endpoint-item:last-child {
  border-bottom: none;
}

.endpoint-name {
  display: block;
  font-weight: 600;
  color: #c0c2d8;
  margin-bottom: 2px;
}

.endpoint-addrs {
  display: block;
  color: #9ca0b8;
  font-size: 0.8rem;
}

.empty-small {
  padding: 8px 0;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.btn-danger {
  color: #ef4444;
}
</style>
