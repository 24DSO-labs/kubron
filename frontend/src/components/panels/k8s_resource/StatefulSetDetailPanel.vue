<template>
  <div v-if="statefulset" class="panel-wrapper">
  <K8sResourceDetailBase :resource="statefulset" resource-icon="" :custom-tabs="customTabs" @close="$emit('close')"
      @edit="$emit('edit')" @delete="$emit('delete')" @namespaceClick="onNamespaceClick">
      <template #actions>
        <!-- Restart -->
        <button class="btn-action" title="Restart statefulset"
          @click="handleContextAction('restart', statefulset)">
          <img :src="restartIcon" alt="Restart" class="btn-icon" />
        </button>

        <!-- Logs -->
        <button class="btn-action" title="View logs" @click="handleContextAction('logs', statefulset)">
          <img :src="logsIcon" alt="Logs" class="btn-icon" />
        </button>
      </template>

      <template #properties>
        <div class="sts-properties">
          <!-- Labels -->
          <div class="property-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
            <span class="property-label">
              Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
            </span>
            <div v-if="!labelsExpanded" class="property-value">{{ Object.keys(statefulset.labels || {}).length }} Labels</div>
            <div v-else class="property-details">
              <div v-for="[key, value] in Object.entries(statefulset.labels || {})" :key="key" class="detail-item">
                <span class="detail-key">{{ key }}</span>
                <span class="detail-value">{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- Annotations -->
          <div class="property-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
            <span class="property-label">
              Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
            </span>
            <div v-if="!annotationsExpanded" class="property-value">{{ Object.keys(statefulset.annotations || {}).length }} Annotations</div>
            <div v-else class="property-details">
              <div v-for="[key, value] in Object.entries(statefulset.annotations || {})" :key="key" class="detail-item">
                <span class="detail-key">{{ key }}</span>
                <span class="detail-value">{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- Selector -->
          <div class="property-row">
            <span class="property-label">Selector</span>
            <span class="property-value">{{ statefulset.selector || '—' }}</span>
          </div>

          <!-- Replicas -->
          <div class="property-row">
            <span class="property-label">Replicas</span>
            <span class="property-value">{{ statefulset.replicas || 0 }} / {{ statefulset.readyReplicas || 0 }}</span>
          </div>

          <!-- Images -->
          <div class="property-row">
            <span class="property-label">Images</span>
            <div class="property-value images-list">
              <div v-for="(img, idx) in allContainerImages" :key="idx" class="image-item">{{ img }}</div>
              <span v-if="allContainerImages.length === 0">—</span>
            </div>
          </div>

          <!-- Strategy Type -->
          <div class="property-row">
            <span class="property-label">Strategy Type</span>
            <span class="property-value">{{ statefulset.strategy || 'RollingUpdate' }}</span>
          </div>

          <!-- Pod Status -->
          <div class="property-row">
            <span class="property-label">Pod Status</span>
            <span class="property-value">{{ podStatusSummary }}</span>
          </div>
        </div>
      </template>

      <template #tab-pods>
        <div class="pods-section">
          <div v-if="statefulsetPods && statefulsetPods.length" class="pods-container">
            <table class="pods-table">
              <thead>
                <tr>
                  <th class="col-name">Name</th>
                  <th class="col-node">Node</th>
                  <th class="col-namespace">Namespace</th>
                  <th class="col-ready">Ready</th>
                  <th class="col-cpu">CPU</th>
                  <th class="col-memory">Memory</th>
                  <th class="col-status">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="pod in statefulsetPods" :key="`${pod.namespace}-${pod.name}`" class="pod-row"
                  @click="selectPod(pod)">
                  <td class="col-name">{{ pod.name }}</td>
                  <td class="col-node">{{ pod.node || '—' }}</td>
                  <td class="col-namespace">{{ pod.namespace }}</td>
                  <td class="col-ready">{{ pod.ready }}</td>
                  <td class="col-cpu">{{ formatCpu(podMetricsMap[pod.name]?.totalCpu) }}</td>
                  <td class="col-memory">{{ formatMemory(podMetricsMap[pod.name]?.totalMem) }}</td>
                  <td class="col-status">
                    <span class="status-badge" :class="'status-' + (pod.status || 'unknown').toLowerCase()">
                      {{ pod.status || 'Unknown' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="empty-state-text">No pods available</div>
        </div>
      </template>

      <template #events>
        <div class="empty-section">No events available</div>
      </template>
    </K8sResourceDetailBase>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import type { PodInfo, ConditionInfo } from '../../../types'
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import restartIcon from '../../../assets/icons/restart.svg'
import logsIcon from '../../../assets/icons/logs.svg'
import { useK8s } from '../../../composables/useK8s'
import { useKubStatefulSetActions } from '../../../composables/KubStatefulSetActions'

const props = defineProps<{ statefulset: any; clusterName?: string }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'pod-selected', pod: PodInfo): void
  (e: 'open-resource', data: { kind: string; name: string; namespace?: string }): void
}>()

const { pods, podMetrics } = useK8s()

interface DetailTab {
  id: string
  label: string
}

const customTabs: DetailTab[] = [
  { id: 'pods', label: 'Pods' }
]

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const statefulsetPods = ref<any[]>([])
const podMetricsMap = ref<Record<string, any>>({})

// Используем композабл для действий
const selectedResource = computed(() => props.statefulset)

const resourceActions = {
  restart: (resource: any) => {
    emit('restart', resource.namespace, resource.name)
  },
  logs: (resource: any) => {
    emit('logs', resource.namespace, resource.name)
  }
}

const k8sActions = {
  showYAML: (kind: string, namespace: string, name: string) => {
    emit('edit')
  },
  refreshData: () => {},
  deleteResource: (namespace: string, name: string) => {
    emit('delete')
  }
}

const { handleContextAction } = useKubStatefulSetActions(
  selectedResource,
  resourceActions,
  k8sActions
)

const podStatusSummary = computed(() => {
  if (!props.statefulset) return '—'
  const ready = props.statefulset.readyReplicas || 0
  const total = props.statefulset.replicas || 0
  return `Ready: ${ready}/${total}`
})

const allContainerImages = computed(() => {
  const images = new Set<string>()
  if (statefulsetPods.value && statefulsetPods.value.length > 0) {
    statefulsetPods.value.forEach((pod: any) => {
      if (pod.containers && Array.isArray(pod.containers)) {
        pod.containers.forEach((container: any) => {
          if (container.image) {
            images.add(container.image)
          }
        })
      }
    })
  }
  return Array.from(images).sort()
})

function openResource(kind: string, name: string) {
  console.log('StatefulSetDetailPanel.openResource called:', { kind, name, namespace: props.statefulset?.namespace })
  const data = {
    kind,
    name,
    namespace: props.statefulset?.namespace
  }
  console.log('Emitting open-resource:', data)
  emit('open-resource', data)
}

function onNamespaceClick(namespace: string) {
  console.log('StatefulSetDetailPanel.onNamespaceClick called with:', namespace)
  openResource('Namespace', namespace)
}

function selectPod(pod: any): void {
  emit('pod-selected', pod)
}

function formatCpu(value: string | number | undefined): string {
  if (!value) return '—'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '—'
  return num.toFixed(3)
}

function formatMemory(value: string | number | undefined): string {
  if (!value) return '—'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '—'
  const mib = num / (1024 * 1024)
  return mib.toFixed(2) + ' MiB'
}

async function loadStatefulSetPods() {
  if (!props.statefulset) {
    statefulsetPods.value = []
    return
  }

  try {
    const { getPods } = useK8s()
    const allPods = await getPods(props.statefulset.namespace) as PodInfo[]

    const filtered = (allPods || []).filter((pod: PodInfo) => {
      if (pod.namespace !== props.statefulset?.namespace) return false

      if (pod.controlledBy) {
        if (pod.controlledBy.includes(props.statefulset.name)) {
          return true
        }
      }

      if (pod.labels && props.statefulset.selector) {
        try {
          const selectorParts = (props.statefulset.selector || '').split(',').map((s: string) => s.trim())
          const matches = selectorParts.every((part: string) => {
            const [key, value] = part.split('=').map((s: string) => s.trim())
            return pod.labels && pod.labels[key] === value
          })
          return matches
        } catch (e) {
          console.error('Error parsing selector:', e)
          return false
        }
      }

      return false
    })

    statefulsetPods.value = filtered

    const metricsMap: Record<string, any> = {}
    if (podMetrics.value && podMetrics.value.length > 0) {
      podMetrics.value.forEach((metric: any) => {
        if (metric.namespace === props.statefulset?.namespace) {
          metricsMap[metric.name] = metric
        }
      })
    }

    podMetricsMap.value = metricsMap
  } catch (err) {
    console.error('Error loading statefulset pods:', err)
    statefulsetPods.value = []
  }
}

watch(() => props.statefulset, () => {
  loadStatefulSetPods()
}, { immediate: true, deep: true })

watch(() => pods.value, () => {
  loadStatefulSetPods()
}, { deep: true })
</script>

<style scoped>
.sts-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.property-row {
  display: grid;
  grid-template-columns: 150px 1fr;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  align-items: flex-start;
  font-size: 0.875rem;
}

.property-row:last-child {
  border-bottom: none;
}

.property-label {
  font-weight: 600;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.9rem;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 6px;
}

.property-value {
  color: #d4d4d4;
  text-align: left;
  word-break: break-word;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  font-size: 1.2em;
  line-height: 1;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.property-details {
  grid-column: 1 / -1;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border-left: 3px solid #60a5fa;
  margin-top: -8px;
  margin-left: 0;
}

.detail-item {
  display: grid;
  grid-template-columns: 150px 1fr;
  gap: 12px;
  padding: 6px 0;
  font-size: 0.85rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-key {
  color: #60a5fa;
  font-weight: 500;
  word-break: break-all;
}

.detail-value {
  color: #e5e7eb;
  word-break: break-all;
}

.images-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.image-item {
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 0.85rem;
}

.status-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
}

.status-running {
  color: #34d399;
}

.status-pending {
  color: #fbbf24;
}

.status-failed {
  color: #f87171;
}

.status-unknown {
  color: #9ca3af;
}

/* Pods Table Styles */
.pods-section {
  padding: 0 !important;
}

.pods-container {
  width: 100%;
  overflow-x: auto !important;
  scrollbar-color: #35365a #1a1b2e !important;
  scrollbar-width: thin !important;
}

.pods-container::-webkit-scrollbar {
  width: 8px !important;
  height: 8px !important;
}

.pods-container::-webkit-scrollbar-track {
  background: #1a1b2e !important;
}

.pods-container::-webkit-scrollbar-thumb {
  background: #35365a !important;
  border-radius: 4px !important;
}

.pods-container::-webkit-scrollbar-thumb:hover {
  background: #45466a !important;
}

.pods-container::-webkit-scrollbar-corner {
  background: #1a1b2e !important;
}

.pods-table {
  width: 100%;
  border-collapse: collapse;
  background: #1a1b2e;
  color: #d4d4d4;
  font-size: 0.9rem;
}

.pods-table thead {
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid #2a2b40;
  position: sticky;
  top: 0;
  z-index: 10;
}

.pods-table th {
  padding: 10px 16px !important;
  text-align: left !important;
  font-weight: 600 !important;
  border: none !important;
  white-space: nowrap !important;
  text-transform: uppercase !important;
  font-size: 0.75rem !important;
  letter-spacing: 0.5px !important;
  color: #868d8a !important;
  background: rgba(255, 255, 255, 0.02) !important;
}

.pods-table tbody tr {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background 0.15s;
  cursor: pointer;
}

.pods-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}

.pods-table td {
  padding: 12px 16px;
  border: none;
  text-align: left;
}

.pod-row {
  cursor: pointer;
}

.col-name {
  flex: 1;
  min-width: 120px;
  color: #60a5fa;
}

.col-node {
  width: 100px;
  min-width: 100px;
  color: #d4d4d4;
}

.col-namespace {
  width: 80px;
  min-width: 80px;
  color: #6b6d8a;
}

.col-ready {
  width: 50px;
  min-width: 50px;
  text-align: center;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-cpu {
  width: 60px;
  min-width: 60px;
  text-align: right;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-memory {
  width: 80px;
  min-width: 80px;
  text-align: right;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-status {
  width: 80px;
  min-width: 80px;
  text-align: left;
}

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: none;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  display: flex;
  align-items: center;
  justify-content: center;
  min-width: 32px;
  height: 32px;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.05);
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.empty-state-text {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #6b6d8a;
  font-size: 0.9rem;
}

.btn-icon {
  width: 24px;
  height: 24px;
  display: inline-block;
  vertical-align: middle;
  filter: invert(1);
  opacity: 0.85;
  transition: opacity 0.15s;
}

.btn-action:hover .btn-icon {
  opacity: 1;
}

.cursor-pointer {
  cursor: pointer;
}
</style>
