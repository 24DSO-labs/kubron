<template>
  <div v-if="deployment" class="panel-wrapper">
    <K8sResourceDetailBase
      :resource="deployment"
      resource-icon=""
      @close="$emit('close')"
    >
    <template #actions>
      <!-- Scale -->
      <button class="btn-action" title="Scale deployment" @click="$emit('scale', deployment)">
        <img :src="scaleIcon" alt="Scale" class="btn-icon"/>
      </button>

      <!-- Restart -->
      <button class="btn-action" title="Restart deployment" @click="$emit('restart', deployment.namespace, deployment.name)">
        <img :src="restartIcon" alt="Restart" class="btn-icon"/>
      </button>

      <!-- Logs -->
      <button class="btn-action" title="View logs" @click="$emit('logs', deployment.namespace, deployment.name)">
        <img :src="logsIcon" alt="Logs" class="btn-icon"/>
      </button>
    </template>

    <template #properties>
      <div class="dep-properties">
        <div class="meta-row">
          <span class="meta-label">Replicas:</span>
          {{ deployment.replicas }} / {{ deployment.desiredReplicas }}
        </div>
        <div class="meta-row">
          <span class="meta-label">Selector:</span>
          <div class="label-list">
            <span v-for="(v, k) in deployment.selector" :key="k" class="label-pair">
              {{ k }}={{ v }}
            </span>
          </div>
        </div>
        <div class="meta-row">
          <span class="meta-label">Template:</span>
          <div class="label-list">
            <span v-for="(v, k) in deployment.template?.labels" :key="k" class="label-pair">
              {{ k }}={{ v }}
            </span>
          </div>
        </div>
        <div class="meta-row">
          <span class="meta-label">Strategy:</span>
          {{ deployment.strategy?.type }}
        </div>
      </div>
    </template>

    <template #events>
      <div class="empty-section">No events available</div>
    </template>
    </K8sResourceDetailBase>
  </div>
</template>

<script lang="ts" setup>
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import scaleIcon from '../../../assets/icons/scale.svg'
import restartIcon from '../../../assets/icons/restart.svg'
import logsIcon from '../../../assets/icons/logs.svg'

defineProps<{ deployment: any }>()
defineEmits<{
  (e: 'close'): void
  (e: 'scale', dep: any): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'edit'): void
  (e: 'delete'): void
}>()
</script>

<style scoped>
.dep-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  gap: 12px;
  font-size: 0.875rem;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
}

.label-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}

.label-pair {
  display: inline-block;
  padding: 4px 8px;
  background: rgba(96, 165, 250, 0.15);
  color: #60a5fa;
  border-radius: 3px;
  font-size: 0.8rem;
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

.btn-action.btn-danger {
  color: #ef4444;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
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

.btn-action.btn-danger .btn-icon {
  filter: invert(1);
  opacity: 0.9;
}

.btn-action.btn-danger:hover .btn-icon {
  opacity: 1;
}
</style>
