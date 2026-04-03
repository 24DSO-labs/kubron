<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>📜 Deploy Revisions</h3>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>

    <div class="panel-body">
      <div v-if="sortedRevisions && sortedRevisions.length" class="revisions-container">
        <table class="revisions-table">
          <thead>
            <tr>
              <th class="col-revision">REVISION</th>
              <th class="col-name">NAME</th>
              <th class="col-replicas">REPLICAS</th>
              <th class="col-ready">READY</th>
              <th class="col-images">IMAGES</th>
              <th class="col-actions"></th>
            </tr>
          </thead>
          <tbody>
            <tr v-for="(revision, idx) in sortedRevisions" :key="idx" class="revision-row">
              <td class="col-revision">
                <span class="revision-badge">{{ revision.revision || idx + 1 }}</span>
              </td>
              <td class="col-name">{{ revision.name }}</td>
              <td class="col-replicas">{{ revision.replicas }}</td>
              <td class="col-ready">{{ revision.ready }}</td>
              <td class="col-images">
                <span v-if="revision.images" class="image-tag">{{ revision.images }}</span>
                <span v-else class="image-tag">—</span>
              </td>
              <td class="col-actions">
                <div class="action-menu-wrapper">
                  <button class="btn-menu" @click="openContextMenu(idx, $event)">⋮</button>
                  <div v-if="activeMenuIdx === idx" class="context-menu" :style="menuStyle">
                    <button class="menu-item" @click="closeMenu(); $emit('show-details', revision)">
                      📋 Show Details
                    </button>
                    <button class="menu-item" @click="closeMenu(); $emit('scale', revision)">
                      ⬍ Scale
                    </button>
                    <button class="menu-item" @click="closeMenu(); $emit('logs', revision)">
                      📝 Logs
                    </button>
                    <button class="menu-item" @click="closeMenu(); $emit('edit', revision)">
                      ✏️ Edit
                    </button>
                    <button class="menu-item" @click="closeMenu(); $emit('delete', revision)">
                      🗑️ Delete
                    </button>
                    <div class="menu-separator"></div>
                    <button class="menu-item menu-rollback" @click="closeMenu(); $emit('rollback', revision.revision)">
                      ↩️ Rollback
                    </button>
                  </div>
                </div>
              </td>
            </tr>
          </tbody>
        </table>
      </div>
      <div v-else class="empty-state">
        <p>No revisions available</p>
      </div>
    </div>

    <!-- Click overlay to close menu -->
    <div v-if="activeMenuIdx !== -1" class="menu-overlay" @click="closeMenu()"></div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

interface Revision {
  revision: string | number
  name: string
  replicas: number
  ready: number
  images?: string
  age?: string
}

interface Props {
  revisions: Revision[]
}

const props = defineProps<Props>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'rollback', rev: string | number): void
  (e: 'show-details', revision: Revision): void
  (e: 'scale', revision: Revision): void
  (e: 'logs', revision: Revision): void
  (e: 'edit', revision: Revision): void
  (e: 'delete', revision: Revision): void
}>()

const activeMenuIdx = ref(-1)
const menuStyle = ref<Record<string, string>>({})

function openContextMenu(idx: number, event: MouseEvent) {
  event.stopPropagation()
  activeMenuIdx.value = activeMenuIdx.value === idx ? -1 : idx

  if (activeMenuIdx.value === idx) {
    const button = event.target as HTMLElement
    const rect = button.getBoundingClientRect()
    menuStyle.value = {
      position: 'fixed',
      top: `${rect.bottom + 4}px`,
      right: `${window.innerWidth - rect.right}px`,
      zIndex: '1000',
    }
  }
}

function closeMenu() {
  activeMenuIdx.value = -1
}

const sortedRevisions = computed(() => {
  return [...props.revisions].sort((a, b) => {
    const revA = parseInt(String(a.revision || 0), 10)
    const revB = parseInt(String(b.revision || 0), 10)
    return revB - revA
  })
})
</script>

<style scoped>
.detail-panel {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #1a1b2e;
  border: 1px solid #2a2b40;
  border-radius: 6px;
  overflow: hidden;
}

.detail-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  border-bottom: 1px solid #2a2b40;
  background: #1f2139;
  gap: 12px;
}

.detail-header h3 {
  margin: 0;
  flex: 1;
  color: #d4d4d4;
  font-size: 1rem;
  font-weight: 600;
}

.btn-close {
  padding: 4px 8px;
  background: transparent;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.95rem;
  transition: all 0.15s;
}

.btn-close:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #5a5c7a;
  color: white;
}

.panel-body {
  flex: 1;
  overflow-y: auto;
  padding: 0;
}

.revisions-container {
  width: 100%;
  overflow-x: auto;
}

.revisions-table {
  width: 100%;
  border-collapse: collapse;
  background: #1a1b2e;
  color: #d4d4d4;
  font-size: 0.9rem;
}

.revisions-table thead {
  background: #1f2139;
  border-bottom: 1px solid #2a2b40;
  position: sticky;
  top: 0;
  z-index: 10;
}

.revisions-table th {
  padding: 12px 16px;
  text-align: left;
  font-weight: 600;
  color: #a0a2c0;
  border: none;
  white-space: nowrap;
  font-size: 0.85rem;
  letter-spacing: 0.5px;
  text-transform: uppercase;
}

.revisions-table tbody tr {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background 0.15s;
}

.revisions-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}

.revisions-table td {
  padding: 12px 16px;
  border: none;
  text-align: left;
  vertical-align: middle;
}

.revision-row {
  cursor: default;
}

.col-revision {
  width: 80px;
  min-width: 80px;
}

.revision-badge {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 32px;
  height: 32px;
  background: rgba(59, 130, 246, 0.2);
  border: 1px solid #3b82f6;
  border-radius: 4px;
  color: #60a5fa;
  font-weight: 600;
  font-size: 0.9rem;
}

.col-name {
  flex: 1;
  min-width: 150px;
  color: #d4d4d4;
  word-break: break-word;
}

.col-replicas {
  width: 80px;
  min-width: 80px;
  text-align: center;
  color: #d4d4d4;
}

.col-ready {
  width: 80px;
  min-width: 80px;
  text-align: center;
  color: #d4d4d4;
}

.col-images {
  width: 150px;
  min-width: 150px;
}

.image-tag {
  display: inline-block;
  padding: 4px 8px;
  background: rgba(59, 130, 246, 0.1);
  border: 1px solid #3b82f6;
  border-radius: 3px;
  color: #60a5fa;
  font-size: 0.85rem;
  max-width: 100%;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.col-actions {
  width: 40px;
  min-width: 40px;
  text-align: center;
  padding: 12px 8px;
}

.action-menu-wrapper {
  position: relative;
  display: inline-block;
}

.btn-menu {
  padding: 4px 6px;
  background: transparent;
  border: 1px solid transparent;
  border-radius: 3px;
  color: #6b6d8a;
  cursor: pointer;
  font-size: 0.95rem;
  transition: all 0.15s;
}

.btn-menu:hover {
  background: rgba(255, 255, 255, 0.1);
  border-color: #3b3d5c;
  color: #d4d4d4;
}

.context-menu {
  background: #22233d;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  min-width: 160px;
  z-index: 1000;
}

.menu-item {
  display: block;
  width: 100%;
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-radius: 0;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.85rem;
  text-align: left;
  white-space: nowrap;
  transition: all 0.15s;
}

.menu-item:first-child {
  border-top-left-radius: 3px;
  border-top-right-radius: 3px;
}

.menu-item:last-child {
  border-bottom-left-radius: 3px;
  border-bottom-right-radius: 3px;
}

.menu-item:hover {
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
}

.menu-separator {
  height: 1px;
  background: #3b3d5c;
  margin: 4px 0;
}

.menu-rollback {
  color: #fbbf24;
  font-weight: 600;
}

.menu-rollback:hover {
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 999;
}

.empty-state {
  display: flex;
  align-items: center;
  justify-content: center;
  height: 200px;
  color: #6b6d8a;
  font-size: 0.9rem;
}

.empty-state p {
  margin: 0;
}
</style>
