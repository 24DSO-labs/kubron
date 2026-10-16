<template>
  <div class="pf-control">
    <div v-if="active" class="pf-active">
      <span class="pf-label">→ localhost:{{ active.localPort }}</span>
      <button class="pf-btn-stop" @click="stop">Stop</button>
    </div>
    <template v-else>
      <button class="pf-btn" @click="showModal = true">Forward...</button>
    </template>

    <div v-if="showModal" class="pf-modal-overlay" @click="showModal = false">
      <div class="pf-modal" @click.stop>
        <div class="pf-modal-head">
          <span>Port Forward</span>
          <button class="pf-close" @click="showModal = false">✕</button>
        </div>
        <div class="pf-modal-body">
          <div class="pf-row">
            <label>Target port</label>
            <span class="pf-target">{{ port }}</span>
          </div>
          <div class="pf-row">
            <label>Local port</label>
            <input v-model.number="localPort" type="number" min="1" max="65535" class="pf-input" @keyup.enter="start" />
          </div>
        </div>
        <div class="pf-modal-foot">
          <button class="pf-btn-secondary" @click="showModal = false">Cancel</button>
          <button class="pf-btn-primary" @click="start" :disabled="loading">{{ loading ? '...' : 'Forward' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'
import { useK8s } from '../../../composables/useK8s'
import { callApp } from '../../../resources/wails'

const props = defineProps<{
  namespace: string
  resourceName: string
  port: number
  resourceType: 'pod' | 'service'
}>()

const k8s = useK8s()
const showModal = ref(false)
const localPort = ref(props.port + 10000)
const loading = ref(false)

const active = computed(() => {
  return k8s.portForwards.value.find(
    (s: any) =>
      s.namespace === props.namespace &&
      s[props.resourceType] === props.resourceName &&
      s.podPort === props.port,
  )
})

async function start() {
  loading.value = true
  try {
    if (props.resourceType === 'pod') {
      await callApp('StartPortForward', props.namespace, props.resourceName, localPort.value, props.port)
    } else {
      await callApp('StartServicePortForward', props.namespace, props.resourceName, localPort.value, props.port)
    }
    await k8s.refreshPortForwards()
    showModal.value = false
  } catch (e: any) {
    alert('Port forward error: ' + (e?.message || e))
  } finally {
    loading.value = false
  }
}

async function stop() {
  if (!active.value) return
  try {
    await callApp('StopPortForward', active.value.id)
    await k8s.refreshPortForwards()
  } catch (e: any) {
    alert('Stop forward error: ' + (e?.message || e))
  }
}
</script>

<style scoped>
.pf-control { display: inline-flex; align-items: center; }
.pf-btn { padding: 2px 8px; background: transparent; border: 1px solid #3b3d5c; border-radius: 4px; color: #d4d4d4; cursor: pointer; font-size: 0.75rem; }
.pf-btn:hover { background: rgba(59,130,246,0.2); border-color: #60a5fa; color: #60a5fa; }
.pf-active { display: flex; align-items: center; gap: 6px; }
.pf-label { color: #34d399; font-size: 0.75rem; font-weight: 500; }
.pf-btn-stop { padding: 2px 8px; background: transparent; border: 1px solid #ef4444; border-radius: 4px; color: #ef4444; cursor: pointer; font-size: 0.75rem; }
.pf-btn-stop:hover { background: rgba(239,68,68,0.2); }

.pf-modal-overlay { position: fixed; inset: 0; background: rgba(0,0,0,0.5); display: flex; align-items: center; justify-content: center; z-index: 1000; }
.pf-modal { background: #1e1f35; border: 1px solid #3b3d5c; border-radius: 8px; width: 90%; max-width: 320px; overflow: hidden; }
.pf-modal-head { display: flex; align-items: center; justify-content: space-between; padding: 12px 16px; border-bottom: 1px solid #2a2b40; font-weight: 600; }
.pf-close { background: transparent; border: none; color: #9ca0b8; cursor: pointer; font-size: 1rem; }
.pf-modal-body { padding: 16px; display: flex; flex-direction: column; gap: 12px; }
.pf-row { display: grid; grid-template-columns: auto 1fr; align-items: center; gap: 12px; }
.pf-row label { color: #9ca0b8; font-size: 0.85rem; font-weight: 500; }
.pf-target { color: #60a5fa; font-weight: 600; font-family: monospace; }
.pf-input { padding: 6px 10px; background: #1a1b2e; border: 1px solid #3b3d5c; border-radius: 4px; color: white; font-size: 0.9rem; width: 100%; }
.pf-input:focus { outline: none; border-color: #3b82f6; }
.pf-modal-foot { display: flex; justify-content: flex-end; gap: 8px; padding: 12px 16px; border-top: 1px solid #2a2b40; }
.pf-btn-secondary { padding: 6px 12px; background: transparent; border: 1px solid #3b3d5c; border-radius: 4px; color: #9ca0b8; cursor: pointer; font-size: 0.85rem; }
.pf-btn-primary { padding: 6px 12px; background: #3b82f6; border: none; border-radius: 4px; color: white; cursor: pointer; font-size: 0.85rem; }
.pf-btn-primary:hover:not(:disabled) { background: #2563eb; }
.pf-btn-primary:disabled { opacity: 0.6; cursor: not-allowed; }
</style>
