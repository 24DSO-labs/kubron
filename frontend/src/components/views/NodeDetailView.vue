<template>
  <K8sResourceDetailBase
    :resource="node"
    resource-icon="🖥️"
    :custom-tabs="customTabs"
    @close="emit('close')"
  >
    <!-- Actions -->
    <template #actions>
      <!-- Edit YAML -->
      <button class="btn-action" title="Edit YAML" @click="emit('edit')">✏️</button>
    </template>

    <!-- Properties tab content -->
    <template #properties>
      <div class="node-properties">
        <!-- Node Metrics (Collapsible) -->
        <div v-if="clusterName" class="metrics-section">
          <div class="metrics-header" @click="metricsExpanded = !metricsExpanded">
            <span class="metrics-header-title">Metrics</span>
            <span class="metrics-toggle" :class="{ collapsed: !metricsExpanded }"></span>
          </div>
          <div v-if="metricsExpanded" class="metrics-content">
            <NodeMetricsPanel :cluster-name="clusterName" :node-name="node.name" />
          </div>
        </div>

        <!-- Status -->
        <div class="meta-row">
          <span class="meta-label">Status:</span>
          <span class="status-badge" :class="statusClass(node.status)">{{ node.status }}</span>
        </div>

        <!-- Role -->
        <div class="meta-row">
          <span class="meta-label">Role:</span>
          <span>{{ node.role || '—' }}</span>
        </div>

        <!-- Kubernetes Version -->
        <div class="meta-row">
          <span class="meta-label">Kubernetes Version:</span>
          <span>{{ node.kubeVersion || '—' }}</span>
        </div>

        <!-- Container Runtime -->
        <div class="meta-row">
          <span class="meta-label">Container Runtime:</span>
          <span>{{ node.containerRuntime || '—' }}</span>
        </div>

        <!-- OS Image -->
        <div class="meta-row">
          <span class="meta-label">OS Image:</span>
          <span>{{ node.osImage || '—' }}</span>
        </div>

        <!-- Architecture -->
        <div class="meta-row">
          <span class="meta-label">Architecture:</span>
          <span>{{ node.architecture || '—' }}</span>
        </div>

        <!-- Internal IP -->
        <div class="meta-row">
          <span class="meta-label">Internal IP:</span>
          <span class="node-ip">{{ node.internalIP || '—' }}</span>
        </div>

        <!-- External IP -->
        <div class="meta-row">
          <span class="meta-label">External IP:</span>
          <span class="node-ip">{{ node.externalIP || '—' }}</span>
        </div>

        <!-- Pod CIDR -->
        <div class="meta-row">
          <span class="meta-label">Pod CIDR:</span>
          <span>{{ node.podCIDR || '—' }}</span>
        </div>

        <!-- Labels (expandable) -->
        <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
          <span class="meta-label">
            Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
          </span>
          <div class="meta-content">
            <span v-if="!labelsExpanded">{{ Object.keys(node.labels || {}).length }} Labels</span>
            <div v-else class="labels-list">
              <span v-for="[key, value] in Object.entries(node.labels || {})" :key="key" class="label-pair">
                <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
              </span>
            </div>
          </div>
        </div>

        <!-- Annotations (expandable) -->
        <div class="meta-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
          <span class="meta-label">
            Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
          </span>
          <div class="meta-content">
            <span v-if="!annotationsExpanded">{{ Object.keys(node.annotations || {}).length }} Annotations</span>
            <div v-else class="labels-list">
              <span v-for="[key, value] in Object.entries(node.annotations || {})" :key="key" class="label-pair">
                <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
              </span>
            </div>
          </div>
        </div>

        <!-- Taints (expandable) -->
        <div class="meta-row cursor-pointer" @click="taintsExpanded = !taintsExpanded">
          <span class="meta-label">
            Taints <span class="expand-icon" :class="{ expanded: taintsExpanded }">›</span>
          </span>
          <div class="meta-content">
            <span v-if="!taintsExpanded">{{ node.taints?.length || 0 }}</span>
            <div v-else class="taints-list">
              <span v-for="(t, i) in node.taints" :key="i" class="taint-item">
                <span class="taint-key">{{ t.key }}</span>
                <span v-if="t.value" class="taint-sep">=</span>
                <span v-if="t.value" class="taint-value">{{ t.value }}</span>
                <span class="taint-effect">:{{ t.effect }}</span>
              </span>
            </div>
          </div>
        </div>

        <!-- Allocatable Resources -->
        <div class="meta-row cursor-pointer" @click="resourcesExpanded = !resourcesExpanded">
          <span class="meta-label">
            Allocatable Resources <span class="expand-icon" :class="{ expanded: resourcesExpanded }">›</span>
          </span>
          <div class="meta-content">
            <span v-if="!resourcesExpanded">Configured</span>
          </div>
        </div>
        <div v-if="resourcesExpanded" class="resources-table">
          <div class="resource-row">
            <span class="resource-label">CPU:</span>
            <span class="resource-value">{{ node.allocatableCPU || '—' }}</span>
          </div>
          <div class="resource-row">
            <span class="resource-label">Memory:</span>
            <span class="resource-value">{{ node.allocatableMemory || '—' }}</span>
          </div>
          <div class="resource-row">
            <span class="resource-label">Pods:</span>
            <span class="resource-value">{{ node.allocatablePods || '—' }}</span>
          </div>
          <div class="resource-row">
            <span class="resource-label">Storage:</span>
            <span class="resource-value">{{ node.allocatableStorage || '—' }}</span>
          </div>
        </div>

        <!-- Conditions -->
        <div class="meta-row cursor-pointer" @click="conditionsExpanded = !conditionsExpanded">
          <span class="meta-label">
            Conditions <span class="expand-icon" :class="{ expanded: conditionsExpanded }">›</span>
          </span>
          <div class="meta-content">
            <span v-if="!conditionsExpanded">{{ node.conditions?.length || 0 }}</span>
          </div>
        </div>
        <div v-if="conditionsExpanded" class="conditions-table">
          <div v-for="(c, i) in node.conditions" :key="i" class="condition-row">
            <span class="condition-type">{{ c.type }}</span>
            <span class="condition-status" :class="`status-${c.status.toLowerCase()}`">{{ c.status }}</span>
            <span v-if="c.message" class="condition-message">{{ c.message }}</span>
          </div>
        </div>
      </div>
    </template>

    <!-- Events tab (empty placeholder) -->
    <template #events>
      <div class="empty-section">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'
import K8sResourceDetailBase from '../panels/K8sResourceDetailBase.vue'
import NodeMetricsPanel from '../metrics/NodeMetricsPanel.vue'

const props = defineProps<{
  node: any
  statusClass: (s: string) => string
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
}>()

// ═══════ Properties expand state ═══════

const metricsExpanded = ref(true)
const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const taintsExpanded = ref(false)
const resourcesExpanded = ref(false)
const conditionsExpanded = ref(false)

// ═══════ Custom tabs ═══════

const customTabs = computed(() => [])
</script>

<style scoped>
/* ═══════ Node Properties ═══════ */

.node-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 8px;
  gap: 12px;
  text-align: left;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-content {
  margin: 0;
  padding: 0;
  text-align: left;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  margin-left: 4px;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.labels-list,
.taints-list {
  font-size: 0.875rem;
  margin: 0;
  padding: 0;
}

.label-pair,
.taint-item {
  display: block;
  overflow-wrap: break-word;
  margin-bottom: 4px;
}

.label-key,
.taint-key {
  color: #60a5fa;
  font-weight: 500;
}

.label-value,
.taint-value {
  color: #e5e7eb;
}

.taint-sep {
  color: #6b6d8a;
}

.taint-effect {
  color: #fbbf24;
  margin-left: 4px;
}

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-radius: 3px;
  font-size: 0.8rem;
  font-weight: 600;
}

.node-ip {
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  background: rgba(255, 255, 255, 0.04);
  padding: 2px 6px;
  border-radius: 3px;
}

/* ═══════ Resources Table ═══════ */

.resources-table {
  margin-top: 8px;
  margin-bottom: 12px;
  border: 1px solid #2a2b40;
  border-radius: 4px;
  overflow: hidden;
}

.resource-row {
  display: grid;
  grid-template-columns: 100px 1fr;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #2a2b40;
}

.resource-row:last-child {
  border-bottom: none;
}

.resource-label {
  font-weight: 600;
  color: #9ca0b8;
  font-size: 0.85rem;
}

.resource-value {
  color: #d4d4d4;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
}

/* ═══════ Conditions Table ═══════ */

.conditions-table {
  margin-top: 8px;
  margin-bottom: 12px;
}

.condition-row {
  display: grid;
  grid-template-columns: 150px auto 1fr;
  align-items: center;
  padding: 8px 12px;
  border-bottom: 1px solid #2a2b40;
  gap: 12px;
}

.condition-row:last-child {
  border-bottom: none;
}

.condition-type {
  color: #d4d4d4;
  font-weight: 500;
  white-space: nowrap;
}

.condition-status {
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
  text-transform: uppercase;
}

.condition-status.status-true {
  background: rgba(34, 197, 94, 0.2);
  color: #22c55e;
}

.condition-status.status-false {
  background: rgba(239, 68, 68, 0.2);
  color: #ef4444;
}

.condition-status.status-unknown {
  background: rgba(107, 109, 138, 0.2);
  color: #6b6d8a;
}

.condition-message {
  color: #9ca0b8;
  font-size: 0.85rem;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

/* ═══════ Metrics Section ═══════ */

.metrics-section {
  width: 100%;
  margin-bottom: 16px;
  border: 1px solid #2a2b40;
  border-radius: 6px;
  overflow: hidden;
  background: #1a1b2e;
}

.metrics-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(59, 130, 246, 0.05) 100%);
  border-bottom: 1px solid #2a2b40;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s ease;
}

.metrics-header:hover {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0.1) 100%);
}

.metrics-header-title {
  flex: 1;
  font-weight: 600;
  color: #d4d4d4;
  font-size: 0.95rem;
  letter-spacing: 0.3px;
}

.metrics-toggle {
  display: inline-block;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  position: relative;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.metrics-toggle::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 5px;
  height: 9px;
  border-right: 2px solid #6b6d8a;
  border-bottom: 2px solid #6b6d8a;
  transform: translate(-50%, -50%) rotate(-45deg);
  transition: border-color 0.2s ease;
}

.metrics-header:hover .metrics-toggle::before {
  border-color: #a0a2c0;
}

.metrics-toggle.collapsed {
  transform: rotate(-90deg);
}

.metrics-content {
  padding: 16px;
  max-height: 600px;
  overflow-y: auto;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.cursor-pointer {
  cursor: pointer;
}

/* ═══════ Action Buttons ═══════ */

.btn-action {
  padding: 6px 10px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: white;
  cursor: pointer;
  font-size: 1rem;
  transition: all 0.15s;
  white-space: nowrap;
}

.btn-action:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
}
</style>
