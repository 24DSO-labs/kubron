<template>
  <div class="detail-panel">
    <div class="panel-header">
      <div class="header-content">
        <h2>{{ deployment?.name }}</h2>
        <span class="namespace-badge">{{ deployment?.namespace }}</span>
      </div>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

    <div class="panel-body">
      <div v-if="deployment" class="deployment-details">
        <!-- Basic Info -->
        <div class="detail-section">
          <h3>Basic Information</h3>
          <div class="detail-grid">
            <div class="detail-row">
              <span class="detail-label">Ready:</span>
              <span class="detail-value">{{ deployment.readyReplicas }}/{{ deployment.replicas }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Updated:</span>
              <span class="detail-value">{{ deployment.updatedReplicas }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Strategy:</span>
              <span class="detail-value">{{ deployment.strategy }}</span>
            </div>
            <div class="detail-row">
              <span class="detail-label">Age:</span>
              <span class="detail-value">{{ deployment.age }}</span>
            </div>
          </div>
        </div>

        <!-- Images -->
        <div v-if="deployment.images && deployment.images.length" class="detail-section">
          <h3>Container Images</h3>
          <div class="images-list">
            <div v-for="(img, idx) in deployment.images" :key="idx" class="image-item">
              {{ img }}
            </div>
          </div>
        </div>

        <!-- Placeholder for future content -->
        <div class="detail-section">
          <h3>Additional Details</h3>
          <p class="placeholder-text">More details coming soon...</p>
        </div>
      </div>
      <div v-else class="empty-state">
        Loading...
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { DeploymentInfo } from '../../types'

defineProps<{
  deployment: DeploymentInfo | null
}>()

defineEmits<{
  (e: 'close'): void
}>()
</script>

<style scoped>
.detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1b2e;
  border-left: 1px solid #3b3d5c;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #3b3d5c;
  background: #0f1017;
}

.header-content {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
  min-width: 0;
}

.header-content h2 {
  margin: 0;
  font-size: 1.2rem;
  color: white;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.namespace-badge {
  padding: 4px 10px;
  background: rgba(59, 130, 246, 0.2);
  border-radius: 3px;
  color: #60a5fa;
  font-size: 0.85rem;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-close {
  background: transparent;
  border: none;
  color: #9ca0b8;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 4px;
  margin-left: 12px;
  transition: color 0.15s;
}

.btn-close:hover {
  color: white;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 16px;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 100%;
  color: #6b6d8a;
  font-size: 0.95rem;
}

.deployment-details {
  display: flex;
  flex-direction: column;
  gap: 24px;
}

.detail-section {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.detail-section h3 {
  margin: 0;
  color: #d4d4d4;
  font-size: 0.95rem;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  color: #9ca0b8;
}

.detail-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 12px;
}

.detail-row {
  display: flex;
  flex-direction: column;
  gap: 4px;
  padding: 10px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
}

.detail-label {
  color: #9ca0b8;
  font-size: 0.8rem;
  font-weight: 600;
}

.detail-value {
  color: #d4d4d4;
  font-size: 0.9rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.images-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.image-item {
  padding: 10px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  color: #d4d4d4;
  font-size: 0.85rem;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  word-break: break-all;
}

.placeholder-text {
  color: #6b6d8a;
  font-size: 0.9rem;
  margin: 0;
}
</style>
