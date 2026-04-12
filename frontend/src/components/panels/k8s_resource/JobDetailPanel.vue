<template>
  <ResourceDetailsBase 
    v-if="jobData"
    :resource="jobData" 
    :resource-type="'job'"
    resource-icon=""
    :title="jobData.name || 'Job'"
    @close="$emit('close')"
    @edit="handleEdit"
    @delete="handleDelete"
    @namespace-click="handleNamespaceClick"
    @context-action="handleContextAction"
  >
    <template #actions>
      <button class="btn-action btn-retry" title="Retry job" @click.stop="handleRetry">
        <img :src="restartIcon" alt="Retry" class="btn-icon" />
      </button>
    </template>

    <template #properties>
      <div class="resource-properties">
        <div class="property-section">
          <h4 class="section-title">Job Spec</h4>
          <div class="property-grid">
            <div class="property-row">
              <span class="property-label">Parallelism:</span>
              <span class="property-value">{{ jobData.spec?.parallelism ?? 1 }}</span>
            </div>
            <div class="property-row">
              <span class="property-label">Completions:</span>
              <span class="property-value">{{ jobData.spec?.completions ?? 1 }}</span>
            </div>
            <div class="property-row">
              <span class="property-label">Backoff Limit:</span>
              <span class="property-value">{{ jobData.spec?.backoffLimit ?? 6 }}</span>
            </div>
            <div class="property-row">
              <span class="property-label">Active:</span>
              <span class="property-value">{{ jobData.status?.active ?? 0 }}</span>
            </div>
            <div class="property-row">
              <span class="property-label">Succeeded:</span>
              <span class="property-value">{{ jobData.status?.succeeded ?? 0 }}</span>
            </div>
            <div class="property-row">
              <span class="property-label">Failed:</span>
              <span class="property-value">{{ jobData.status?.failed ?? 0 }}</span>
            </div>
          </div>
        </div>

        <div v-if="container" class="property-section">
          <h4 class="section-title">Container</h4>
          <div class="property-grid">
            <div class="property-row" v-if="container.image">
              <span class="property-label">Image:</span>
              <span class="property-value">{{ container.image }}</span>
            </div>
            <div class="property-row" v-if="container.resources?.requests?.cpu || container.resources?.requests?.memory">
              <span class="property-label">Requests:</span>
              <span class="property-value">
                <span v-if="container.resources?.requests?.cpu">{{ formatCpu(container.resources.requests.cpu) }}</span>
                <span v-if="container.resources?.requests?.cpu && container.resources?.requests?.memory">, </span>
                <span v-if="container.resources?.requests?.memory">{{ formatMem(container.resources.requests.memory) }}</span>
              </span>
            </div>
          </div>
        </div>

        <div class="property-section" v-if="jobData.status?.conditions?.length">
          <h4 class="section-title">Conditions</h4>
          <div class="conditions-list">
            <div v-for="condition in jobData.status.conditions" :key="condition.type" :class="['condition-item', getStatusClass(condition.type)]">
              <span class="condition-type">{{ condition.type }}</span>
              <span class="condition-status">{{ condition.status }}</span>
              <span v-if="condition.reason" class="condition-reason">{{ condition.reason }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <template #events>
      <div class="events-list">
        <div v-if="!events?.length" class="empty-state">No events available</div>
        <div v-else v-for="event in events" :key="event.metadata?.uid || event.uid" class="event-item">
          <div class="event-header">
            <span :class="['event-type', getEventClass(event.type)]">{{ event.type }}</span>
            <span class="event-reason">{{ event.reason }}</span>
          </div>
          <div class="event-message">{{ event.message }}</div>
          <div class="event-meta">
            <span>{{ event.metadata?.name || event.name }}</span>
            <span v-if="event.lastTimestamp"> · {{ formatAgeRelative(event.lastTimestamp) }}</span>
          </div>
        </div>
      </div>
    </template>
  </ResourceDetailsBase>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import ResourceDetailsBase from '../ResourceDetailsBase.vue'
import restartIcon from '../../../assets/icons/restart.svg'

interface JobData {
  name: string
  namespace: string
  uid?: string
  creationTimestamp?: string
  parallelism?: number | string
  completions?: number | string
  backoffLimit?: number | string
  active?: number
  succeeded?: number
  failed?: number
  conditions: any[]
  containers: any[]
  metadata?: any
  spec?: any
  status?: any
}

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'namespaceClick', ns: string): void
  (e: 'contextAction', action: string): void
}>()

// Адаптер для поддержки обоих форматов данных (JobInfo и Kubernetes resource)
const jobData = computed<JobData>(() => {
  const r = props.resource
  // Если это Kubernetes resource формат (с metadata)
  if (r.metadata) {
    return {
      name: r.metadata.name,
      namespace: r.metadata.namespace,
      uid: r.metadata.uid,
      creationTimestamp: r.metadata.creationTimestamp,
      parallelism: r.spec?.parallelism,
      completions: r.spec?.completions,
      backoffLimit: r.spec?.backoffLimit,
      active: r.status?.active,
      succeeded: r.status?.succeeded,
      failed: r.status?.failed,
      conditions: r.status?.conditions || [],
      containers: r.spec?.template?.spec?.containers || [],
      metadata: r.metadata,
      spec: r.spec,
      status: r.status
    }
  }
  // Если это JobInfo формат - используем как есть
  // JobInfo имеет поля: name, namespace, completions, succeeded, failed, active, duration, age, status
  return {
    name: r.name,
    namespace: r.namespace,
    uid: r.uid,
    // Для JobInfo age - это строка вида "2d", нужно преобразовать в дату
    creationTimestamp: r.creationTimestamp || r.created || (r.age ? ageToDate(r.age) : ''),
    parallelism: r.parallelism !== undefined ? r.parallelism : 1,
    completions: r.completions !== undefined ? r.completions : 1,
    backoffLimit: r.backoffLimit !== undefined ? r.backoffLimit : 6,
    active: r.active !== undefined ? r.active : 0,
    succeeded: r.succeeded !== undefined ? r.succeeded : 0,
    failed: r.failed !== undefined ? r.failed : 0,
    conditions: r.conditions || [],
    containers: r.containers || [],
    metadata: null,
    spec: null,
    status: null
  }
})

// Helper function to convert age string (e.g., "2d", "3h", "5m") to ISO date string
function ageToDate(age: string): string {
  if (!age || age === '—') return ''
  const now = new Date()
  const match = age.match(/(\d+)([smhdw])/)
  if (!match) return ''
  const value = parseInt(match[1])
  const unit = match[2]
  const diffMs = 
    unit === 's' ? value * 1000 :
    unit === 'm' ? value * 60 * 1000 :
    unit === 'h' ? value * 60 * 60 * 1000 :
    unit === 'd' ? value * 24 * 60 * 60 * 1000 :
    unit === 'w' ? value * 7 * 24 * 60 * 60 * 1000 : 0
  return new Date(now.getTime() - diffMs).toISOString()
}

const container = computed(() => {
  const containers = jobData.value?.containers
  return Array.isArray(containers) && containers.length > 0 ? containers[0] : null
})
// Events пока не используются - заглушка для будущего функционала
const events: any[] = []

const formatCpu = (cpu: string | number): string => {
  if (typeof cpu === 'number') cpu = cpu.toString()
  if (cpu.endsWith('m')) { const m = parseInt(cpu.slice(0, -1)); return m >= 1000 ? (m/1000).toFixed(1) + ' cores' : m + 'm' }
  const c = parseFloat(cpu)
  return isNaN(c) ? cpu : c >= 1 ? c.toFixed(1) + ' cores' : Math.round(c * 1000) + 'm'
}

const formatMem = (mem: string | number): string => {
  if (typeof mem === 'number') mem = mem.toString()
  const U: Record<string,number> = { Ki:1024, Mi:1048576, Gi:1073741824, Ti:1099511627776 }
  for (const [u,m] of Object.entries(U)) if (mem.endsWith(u)) { const v = parseInt(mem.slice(0,-u.length))*m; return fmtBytes(v) }
  return fmtBytes(parseInt(mem))
}

const fmtBytes = (b: number): string => { const U=['B','Ki','Mi','Gi','Ti']; let i=0,v=b; while(v>=1024&&i<U.length-1){v/=1024;i++} return v.toFixed(1)+' '+U[i] }

const getStatusClass = (s: string): string => {
  const m:Record<string,string> = { complete:'status-succeeded', failed:'status-failed', running:'status-running' }
  return m[s?.toLowerCase()] || ''
}

const getEventClass = (t: string): string => t === 'Warning' ? 'event-warning' : 'event-normal'

const formatAgeRelative = (ts: string): string => {
  if (!ts) return '—'
  const d = new Date(ts), now = new Date(), diff = Math.floor((now.getTime() - d.getTime()) / 1000)
  if (diff < 60) return diff + 's'
  const m = Math.floor(diff / 60)
  if (m < 60) return m + 'm'
  const h = Math.floor(m / 60)
  if (h < 24) return h + 'h'
  return Math.floor(h / 24) + 'd'
}

const handleEdit = () => emit('edit')
const handleDelete = () => emit('contextAction', 'delete')
const handleNamespaceClick = (ns: string) => emit('namespaceClick', ns)
const handleContextAction = (action: string) => emit('contextAction', action)
const handleRetry = () => {
  console.log('Retry button clicked')
  emit('contextAction', 'retry')
}
</script>

<style scoped>
.resource-properties { display: flex; flex-direction: column; gap: 20px; }
.property-section { background: rgba(255,255,255,0.02); border-radius: 8px; padding: 16px; border: 1px solid #2a2b40; }
.section-title { margin: 0 0 12px 0; font-size: 0.875rem; font-weight: 600; color: #8b8dc7; text-transform: uppercase; letter-spacing: 0.5px; }
.property-grid { display: flex; flex-direction: column; gap: 8px; }
.property-row { display: flex; justify-content: space-between; padding: 8px 0; border-bottom: 1px solid rgba(255,255,255,0.05); }
.property-row:last-child { border-bottom: none; }
.property-label { color: #868d8a; font-size: 0.875rem; }
.property-value { color: #d4d4d4; font-size: 0.875rem; font-weight: 500; }
.conditions-list { display: flex; flex-direction: column; gap: 8px; }
.condition-item { display: flex; align-items: center; gap: 12px; padding: 10px 12px; background: rgba(255,255,255,0.02); border-radius: 6px; font-size: 0.875rem; }
.condition-type { font-weight: 600; color: #d4d4d4; }
.condition-status { color: #868d8a; }
.condition-reason { color: #6b6d8a; font-size: 0.8rem; }
.status-succeeded .condition-status { color: #10b981; }
.status-failed .condition-status { color: #ef4444; }
.events-list { display: flex; flex-direction: column; gap: 8px; }
.event-item { padding: 12px; background: rgba(255,255,255,0.02); border-radius: 6px; border-left: 3px solid #3b3d5c; }
.event-item.event-warning { border-left-color: #f59e0b; }
.event-header { display: flex; align-items: center; gap: 8px; margin-bottom: 6px; }
.event-type { font-size: 0.75rem; font-weight: 600; padding: 2px 6px; border-radius: 4px; text-transform: uppercase; }
.event-type.event-normal { background: rgba(59,130,246,0.2); color: #60a5fa; }
.event-type.event-warning { background: rgba(245,158,11,0.2); color: #fbbf24; }
.event-reason { font-weight: 600; color: #d4d4d4; font-size: 0.875rem; }
.event-message { color: #a1a1aa; font-size: 0.875rem; margin-bottom: 6px; line-height: 1.4; }
.event-meta { font-size: 0.75rem; color: #6b6d8a; }
.empty-state { text-align: center; color: #6b6d8a; padding: 24px; font-size: 0.875rem; }
</style>
