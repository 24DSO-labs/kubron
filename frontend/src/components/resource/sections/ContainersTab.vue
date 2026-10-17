<template>
  <div>
    <div v-for="c in containers" :key="c.name" class="ct-block">
      <div class="ct-head" @click="toggle(c.name)">
        <span class="ct-status" :class="statusClass(c)" />
        <span class="ct-name">{{ c.name }}</span>
        <span class="ct-image">{{ c.image }}</span>
        <span class="ct-arrow" :class="{ open: !collapsed.has(c.name) }">›</span>
      </div>
      <div v-if="!collapsed.has(c.name)" class="ct-body">
        <div v-if="clusterName" class="rd-block">
          <div class="rd-block-head" @click="toggleMetrics(c.name)">
            <span>Metrics</span>
            <span class="rd-arrow" :class="{ open: !metricsCollapsed.has(c.name) }">›</span>
          </div>
          <div v-if="!metricsCollapsed.has(c.name)" class="rd-block-body">
            <ContainerMetricsViewer :cluster-name="clusterName" :namespace="resource.namespace" :pod-name="resource.name" :container-name="c.name" />
          </div>
        </div>
        <div class="rd-row"><span class="rd-label">Status</span><span class="rd-value"><span class="rd-badge" :class="badgeClass(c.state)">{{ c.state || '—' }}</span></span></div>
        <div class="rd-row"><span class="rd-label">Image</span><span class="rd-value">{{ c.image || '—' }}</span></div>
        <div class="rd-row"><span class="rd-label">ImagePullPolicy</span><span class="rd-value">{{ c.imagePullPolicy || '—' }}</span></div>
        <div v-if="c.ports?.length" class="rd-row">
          <span class="rd-label">Ports</span>
          <div class="rd-value">
            <div v-for="p in c.ports" :key="p.containerPort" class="ct-port">
              <a class="rd-link" href="#" @click.prevent>{{ p.name ? p.name + ': ' : '' }}{{ p.containerPort }}/{{ p.protocol || 'TCP' }}</a>
              <PortForwardControl :namespace="resource.namespace" :resource-name="resource.name" :port="p.containerPort" resource-type="pod" />
            </div>
          </div>
        </div>
        <div v-if="c.environment?.length" class="rd-block">
          <div class="rd-block-head" @click="toggleEnv(c.name)">
            <span>Environment</span>
            <K8sIcon :name="envCollapsed.has(c.name) ? 'view' : 'view-off'" size="sm" />
          </div>
          <div v-if="!envCollapsed.has(c.name)" class="rd-block-body">
            <div v-for="e in c.environment" :key="e.name" class="ct-env">
              <span class="ct-env-name">{{ e.name }}</span>
              <span class="ct-env-sep">:</span>
              <template v-if="isSecret(e.value)">
                <span class="ct-env-value ct-env-secret">{{ maskSecret(e.value) }}</span>
                <button
                  class="ct-env-eye"
                  :title="envRevealed[e.name] ? 'Hide' : 'Reveal'"
                  @click.stop="envRevealed[e.name] = !envRevealed[e.name]"
                >
                  <K8sIcon :name="envRevealed[e.name] ? 'view-off' : 'view'" size="sm" />
                </button>
                <span v-if="envRevealed[e.name]" class="ct-env-value">{{ e.value }}</span>
              </template>
              <span v-else class="ct-env-value">{{ e.value || '—' }}</span>
            </div>
          </div>
        </div>
        <div v-if="c.mounts?.length" class="rd-block">
          <div class="rd-block-head" @click="toggleMounts(c.name)">
            <span>Mounts</span>
            <K8sIcon :name="mountsCollapsed.has(c.name) ? 'close' : 'close'" size="sm" />
          </div>
          <div v-if="!mountsCollapsed.has(c.name)" class="rd-block-body">
            <div v-for="m in c.mounts" :key="m.name" class="ct-mount">
              <span class="ct-mount-path">{{ m.mountPath }}</span>
              <span class="ct-mount-from">from {{ m.name }} <span v-if="m.readOnly" class="ct-mount-ro">(ro)</span><span v-else class="ct-mount-ro">(rw)</span></span>
            </div>
          </div>
        </div>
        <div class="rd-row"><span class="rd-label">Resources</span><span class="rd-value">{{ fmtRes(c.resources?.requests) }}</span></div>
        <div class="rd-row"><span class="rd-label">Limits</span><span class="rd-value">{{ fmtRes(c.resources?.limits) }}</span></div>
        <div v-if="c.resources?.requests" class="rd-row"><span class="rd-label">Requests</span><span class="rd-value">{{ fmtRes(c.resources.requests) }}</span></div>
        <div v-if="c.resources?.limits" class="rd-row"><span class="rd-label">Limits</span><span class="rd-value">{{ fmtRes(c.resources.limits) }}</span></div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, inject, reactive, ref } from 'vue'
import { clusterNameKey } from '../../../resources/context'
import ContainerMetricsViewer from '../../metrics/ContainerMetricsViewer.vue'
import K8sIcon from '../../icons/K8sIcon.vue'
import PortForwardControl from './PortForwardControl.vue'
import { badgeClass } from '../../../resources/status'

const props = defineProps<{ resource: any }>()
const clusterName = inject(clusterNameKey, ref(undefined))
const containers = computed(() => props.resource?.containers ?? [])
const collapsed = ref<Set<string>>(new Set())
const metricsCollapsed = ref<Set<string>>(new Set(containers.value.map((c: any) => c.name)))
const envCollapsed = ref<Set<string>>(new Set(containers.value.map((c: any) => c.name)))
const mountsCollapsed = ref<Set<string>>(new Set(containers.value.map((c: any) => c.name)))
const envRevealed = reactive<Record<string, boolean>>({})

function toggle(name: string) {
  const s = new Set(collapsed.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  collapsed.value = s
}

function toggleMetrics(name: string) {
  const s = new Set(metricsCollapsed.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  metricsCollapsed.value = s
}

function toggleEnv(name: string) {
  const s = new Set(envCollapsed.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  envCollapsed.value = s
}

function toggleMounts(name: string) {
  const s = new Set(mountsCollapsed.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  mountsCollapsed.value = s
}

function isSecret(value: string): boolean {
  return typeof value === 'string' && value.startsWith('secret(')
}

function maskSecret(value: string): string {
  const match = value.match(/^secret\(([^)]+)\)$/)
  return match ? `secret(${match[1].split('/')[0]})` : 'secret(???)'
}
function statusClass(c: any): string {
  if (c.state === 'running' && c.ready) return 'ct-ok'
  if (c.state === 'running') return 'ct-warn'
  if (['CrashLoopBackOff','Error','OOMKilled'].includes(c.state)) return 'ct-err'
  return 'ct-neutral'
}
function fmtRes(r: Record<string,string> | null | undefined): string {
  if (!r || typeof r !== 'object') return '—'
  const entries = Object.entries(r)
  return entries.length ? entries.map(([k,v])=>`${k}: ${v}`).join(', ') : '—'
}
</script>

<style scoped>
.ct-block { margin-bottom: 12px; border: 1px solid #2a2b40; border-radius: 6px; overflow: hidden; background: #1a1b2e; }
.ct-head { display: flex; align-items: center; gap: 10px; padding: 10px 14px; cursor: pointer; user-select: none; font-weight: 600; }
.ct-head:hover { background: rgba(255,255,255,0.02); }
.ct-status { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.ct-ok { background: #4ade80; } .ct-warn { background: #facc15; } .ct-err { background: #ef4444; } .ct-neutral { background: #868d8a; }
.ct-name { color: #d4d4d4; }
.ct-image { color: #868d8a; font-size: 0.8rem; font-weight: 400; flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.ct-arrow { margin-left: auto; transition: transform 0.2s; color: #868d8a; } .ct-arrow.open { transform: rotate(90deg); }
.ct-body { padding: 12px 14px; border-top: 1px solid #2a2b40; display: flex; flex-direction: column; }
.ct-port { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; } .ct-port-name { color: #868d8a; font-size: 0.8rem; }
.ct-env { display: flex; align-items: center; gap: 6px; margin-bottom: 4px; font-size: 0.85rem; }
.ct-env-name { color: #d4d4d4; font-weight: 500; min-width: 0; }
.ct-env-sep { color: #6b6d8a; }
.ct-env-value { color: #e5e7eb; word-break: break-all; }
.ct-env-secret { color: #fbbf24; }
.ct-env-eye {
  display: inline-flex; align-items: center; justify-content: center;
  width: 22px; height: 22px; padding: 0; margin: 0;
  background: transparent; border: none; border-radius: 3px;
  cursor: pointer; color: #868d8a; transition: all 0.15s;
}
.ct-env-eye:hover { background: rgba(255,255,255,0.08); color: #d4d4d4; }
.ct-mount { display: flex; flex-direction: column; gap: 2px; margin-bottom: 8px; font-size: 0.85rem; }
.ct-mount-path { color: #e5e7eb; }
.ct-mount-from { color: #868d8a; font-size: 0.8rem; }
.ct-mount-ro { color: #facc15; font-size: 0.75rem; }
</style>
