<template>
  <div class="ports-section">
    <!-- Ports List -->
    <div v-if="ports && ports.length" class="ports-list">
  <div v-for="p in ports" :key="portKey(p)" class="port-item">
        <div class="port-info">
          <span class="port-number">{{ p.containerPort }}/{{ p.protocol || 'TCP' }}</span>
          <span v-if="p.name" class="port-name">({{ p.name }})</span>
        </div>
        <button 
          class="btn-forward" 
          :disabled="hasActiveForwardForPort(p.containerPort)"
          @click="openForwardModal(p)" 
          :title="hasActiveForwardForPort(p.containerPort) ? 'Port already forwarded' : 'Create port forward'"
        >
          {{ hasActiveForwardForPort(p.containerPort) ? '✓ Active' : 'Forward...' }}
        </button>
      </div>
    </div>
    <div v-else class="ports-empty">None</div>

    <!-- Active Port Forward Sessions for this Pod -->
    <div v-if="activeSessions.length" class="active-sessions">
      <div class="sessions-title">Active Forwards</div>
      <div v-for="session in activeSessions" :key="session.id" class="session-item">
        <div class="session-info">
          <span class="session-local">localhost:{{ session.localPort }}</span>
          <span class="session-arrow">→</span>
          <span class="session-pod">{{ session.podPort }}</span>
        </div>
        <span class="session-status" :class="'status-' + session.status.toLowerCase()">
          {{ session.status }}
        </span>
        <button class="btn-stop" @click="stopPortForward(session.id)" title="Stop port forward">
          Stop
        </button>
      </div>
    </div>

    <!-- Port Forward Modal -->
    <div v-if="modalOpen" class="pf-modal-overlay" @click="closeModal">
      <div class="pf-modal" @click.stop>
        <div class="pf-modal-header">
          <h4>Create Port Forward</h4>
          <button class="pf-modal-close" @click="closeModal">✕</button>
        </div>
        <div class="pf-modal-body">
          <div class="pf-form-row">
            <label>Target Pod Port:</label>
            <span class="pf-pod-port">{{ selectedPort?.containerPort }}/{{ selectedPort?.protocol || 'TCP' }}</span>
          </div>
        <div class="pf-form-row">
          <label for="local-port">Local Port:</label>
          <input
            id="local-port"
            v-model.number="localPort"
            type="number"
            min="1"
            max="65535"
            class="pf-input"
            @keyup.enter="startForward"
          />
        </div>
        <div class="pf-form-row">
          <label for="open-browser">
            <input
              id="open-browser"
              v-model="openBrowser"
              type="checkbox"
              class="pf-checkbox"
            />
            Open in Browser
          </label>
        </div>
        </div>
        <div class="pf-modal-footer">
          <button class="btn btn-secondary" @click="closeModal">Cancel</button>
          <button class="btn btn-primary" @click="startForward" :disabled="loading">
            {{ loading ? '⏳ Creating...' : '▶ Forward' }}
          </button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

const props = defineProps<{
  namespace: string
  podName: string
  ports: any[] | undefined
  portForwards: any[] | undefined
  onStartPortForward?: (ns: string, podName: string, localPort: number, podPort: number) => Promise<void>
  onStopPortForward?: (id: string) => Promise<void>
}>()

const modalOpen = ref(false)
const selectedPort = ref<any | null>(null)
const localPort = ref(8080)
const loading = ref(false)
const openBrowser = ref(true)

function portKey(p: any): string {
  return (p.name || '') + '-' + p.containerPort + '-' + (p.protocol || 'TCP')
}

function openForwardModal(port: any) {
  selectedPort.value = port
  // Generate random port between 10000 and 60000
  localPort.value = Math.floor(Math.random() * 50000) + 10000
  modalOpen.value = true
}

function closeModal() {
  modalOpen.value = false
  selectedPort.value = null
}

async function startForward() {
  if (!selectedPort.value || !props.onStartPortForward) return
  try {
    loading.value = true
    
    // Убедимся что значение порта правильное
    const finalLocalPort = Number(localPort.value)
    const finalPodPort = selectedPort.value.containerPort
    
    console.log('=== PodPortsSection.startForward ===')
    console.log('Input localPort value:', localPort.value)
    console.log('Parsed localPort:', finalLocalPort)
    console.log('podPort:', finalPodPort)
    console.log('About to call onStartPortForward with:')
    console.log({
      namespace: props.namespace,
      pod: props.podName,
      localPort: finalLocalPort,
      podPort: finalPodPort
    })
    
    await props.onStartPortForward(
      props.namespace,
      props.podName,
      finalLocalPort,
      finalPodPort
    )
    
    console.log('Port forward started successfully')
    
    // Открыть в браузере если отмечено
    if (openBrowser.value) {
      const protocol = selectedPort.value.protocol?.toLowerCase() === 'https' ? 'https' : 'http'
      const url = `${protocol}://localhost:${finalLocalPort}`
      console.log('Opening browser:', url)
      setTimeout(() => {
        window.open(url, '_blank')
      }, 500)
    }
    
    closeModal()
  } catch (e: any) {
    console.error('Port forward error:', e)
    alert('Port forward error: ' + (e.message || e))
  } finally {
    loading.value = false
  }
}

async function stopPortForward(id: string) {
  if (!props.onStopPortForward) return
  try {
    await props.onStopPortForward(id)
  } catch (e: any) {
    console.error('Stop port forward error:', e)
  }
}

const activeSessions = computed(() => {
  if (!props.portForwards) return []
  return props.portForwards.filter(
    pf => pf.namespace === props.namespace && pf.pod === props.podName
  )
})

function hasActiveForwardForPort(containerPort: number): boolean {
  return activeSessions.value.some(session => session.podPort === containerPort)
}
</script>

<style scoped>
.ports-section {
  width: 100%;
}

.ports-empty {
  color: #6b6d8a;
  padding: 4px 0;
}

.ports-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.port-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border: 1px solid rgba(255, 255, 255, 0.05);
  transition: all 0.15s;
}

.port-item:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: rgba(255, 255, 255, 0.1);
}

.port-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.port-number {
  color: #60a5fa;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.9rem;
}

.port-name {
  color: #9ca0b8;
  font-size: 0.85rem;
}

.btn-forward {
  padding: 4px 12px;
  background: #3b82f6;
  border: none;
  border-radius: 4px;
  color: white;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-forward:hover:not(:disabled) {
  background: #2563eb;
}

.btn-forward:disabled {
  opacity: 0.6;
  cursor: not-allowed;
  background: #34d399;
}

.active-sessions {
  margin-top: 12px;
  padding-top: 12px;
  border-top: 1px solid rgba(255, 255, 255, 0.1);
}

.sessions-title {
  color: #9ca0b8;
  font-weight: 600;
  font-size: 0.85rem;
  margin-bottom: 8px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.session-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: rgba(52, 211, 153, 0.05);
  border-radius: 4px;
  border: 1px solid rgba(52, 211, 153, 0.2);
  gap: 8px;
}

.session-info {
  display: flex;
  align-items: center;
  gap: 6px;
  flex: 1;
  min-width: 0;
}

.session-local {
  color: #60a5fa;
  font-weight: 500;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  white-space: nowrap;
}

.session-arrow {
  color: #6b6d8a;
  flex-shrink: 0;
}

.session-pod {
  color: #a0a2c0;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.85rem;
  white-space: nowrap;
}

.session-status {
  padding: 2px 8px;
  border-radius: 3px;
  font-size: 0.75rem;
  font-weight: 600;
  white-space: nowrap;
  flex-shrink: 0;
}

.session-status.status-active,
.session-status.status-running {
  background: rgba(52, 211, 153, 0.2);
  color: #34d399;
}

.session-status.status-pending {
  background: rgba(251, 191, 36, 0.2);
  color: #fbbf24;
}

.btn-stop {
  padding: 4px 12px;
  background: rgba(239, 68, 68, 0.2);
  border: 1px solid rgba(239, 68, 68, 0.5);
  border-radius: 4px;
  color: #f87171;
  font-size: 0.8rem;
  font-weight: 600;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
  flex-shrink: 0;
}

.btn-stop:hover {
  background: rgba(239, 68, 68, 0.3);
  border-color: rgba(239, 68, 68, 0.7);
  color: #fca5a5;
}

/* Modal Styles */

.pf-modal-overlay {
  position: fixed;
  top: 0;
  left: 0;
  right: 0;
  bottom: 0;
  background: rgba(0, 0, 0, 0.5);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
}

.pf-modal {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 8px;
  box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5);
  width: 90%;
  max-width: 400px;
  overflow: hidden;
  display: flex;
  flex-direction: column;
}

.pf-modal-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px;
  border-bottom: 1px solid #2a2b40;
  background: #1a1b2e;
}

.pf-modal-header h4 {
  margin: 0;
  color: white;
  font-size: 1rem;
}

.pf-modal-close {
  background: transparent;
  border: none;
  color: #9ca0b8;
  font-size: 1.2rem;
  cursor: pointer;
  padding: 0;
  width: 24px;
  height: 24px;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: color 0.15s;
}

.pf-modal-close:hover {
  color: white;
}

.pf-modal-body {
  padding: 16px;
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pf-form-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: center;
  gap: 12px;
}

.pf-form-row label {
  color: #9ca0b8;
  font-weight: 600;
  font-size: 0.9rem;
  white-space: nowrap;
}

.pf-pod-port {
  color: #60a5fa;
  font-weight: 600;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
}

.pf-input {
  padding: 8px 12px;
  background: #1a1b2e;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  color: white;
  font-size: 0.9rem;
  transition: all 0.15s;
  width: 100%;
  min-width: 0;
}

.pf-input:focus {
  outline: none;
  border-color: #3b82f6;
  box-shadow: 0 0 0 2px rgba(59, 130, 246, 0.1);
}

.pf-checkbox {
  margin-right: 8px;
  cursor: pointer;
  width: 18px;
  height: 18px;
  accent-color: #3b82f6;
}

.pf-modal-footer {
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
  padding: 12px 16px;
  border-top: 1px solid #2a2b40;
  background: #1a1b2e;
}

.btn {
  padding: 8px 16px;
  border-radius: 4px;
  border: none;
  font-weight: 600;
  font-size: 0.9rem;
  cursor: pointer;
  transition: all 0.15s;
  white-space: nowrap;
}

.btn-secondary {
  background: transparent;
  border: 1px solid #3b3d5c;
  color: #9ca0b8;
}

.btn-secondary:hover {
  background: rgba(255, 255, 255, 0.05);
  border-color: #5a5c7a;
  color: #d4d4d4;
}

.btn-primary {
  background: #3b82f6;
  color: white;
}

.btn-primary:hover:not(:disabled) {
  background: #2563eb;
}

.btn-primary:disabled {
  opacity: 0.6;
  cursor: not-allowed;
}
</style>
