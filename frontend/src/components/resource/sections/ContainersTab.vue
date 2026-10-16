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
        <div v-if="clusterName" class="ct-metrics">
          <ContainerMetricsViewer :cluster-name="clusterName" :namespace="resource.namespace" :pod-name="resource.name" :container-name="c.name" />
        </div>
        <div class="ct-row"><span class="ct-label">Status</span><span class="ct-val">{{ c.state || '—' }}</span></div>
        <div class="ct-row"><span class="ct-label">Image</span><span class="ct-val">{{ c.image || '—' }}</span></div>
        <div class="ct-row"><span class="ct-label">ImagePullPolicy</span><span class="ct-val">{{ c.imagePullPolicy || '—' }}</span></div>
        <div v-if="c.ports?.length" class="ct-row">
          <span class="ct-label">Ports</span>
          <div class="ct-val">
            <div v-for="p in c.ports" :key="p.containerPort" class="ct-port">
              <span>{{ p.protocol || 'TCP' }} {{ p.containerPort }}</span>
              <span v-if="p.name" class="ct-port-name">({{ p.name }})</span>
              <PortForwardControl :namespace="resource.namespace" :resource-name="resource.name" :port="p.containerPort" resource-type="pod" />
            </div>
          </div>
        </div>
        <div v-if="c.environment?.length" class="ct-row">
          <span class="ct-label">Environment</span>
          <div class="ct-val">
            <div v-for="e in c.environment" :key="e.name" class="ct-env">
              <span class="ct-env-name">{{ e.name }}</span><span class="ct-env-sep">=</span><span class="ct-env-value">{{ e.value || e.valueFrom || '—' }}</span>
            </div>
          </div>
        </div>
        <div v-if="c.mounts?.length" class="ct-row">
          <span class="ct-label">Mounts</span>
          <div class="ct-val">
            <div v-for="m in c.mounts" :key="m.name" class="ct-mount">
              <span class="ct-mount-name">{{ m.name }}</span><span class="ct-mount-path">→ {{ m.mountPath }}</span><span v-if="m.readOnly" class="ct-mount-ro">ro</span>
            </div>
          </div>
        </div>
        <div v-if="c.command?.length" class="ct-row"><span class="ct-label">Command</span><span class="ct-val ct-mono">{{ c.command.join(' ') }}</span></div>
        <div v-if="c.args?.length" class="ct-row"><span class="ct-label">Args</span><span class="ct-val ct-mono">{{ c.args.join(' ') }}</span></div>
        <div v-if="c.workingDir" class="ct-row"><span class="ct-label">WorkingDir</span><span class="ct-val ct-mono">{{ c.workingDir }}</span></div>
        <div v-if="c.resources" class="ct-row">
          <span class="ct-label">Resources</span>
          <div class="ct-val">
            <div v-if="c.resources.requests">Requests: {{ fmtRes(c.resources.requests) }}</div>
            <div v-if="c.resources.limits">Limits: {{ fmtRes(c.resources.limits) }}</div>
          </div>
        </div>
        <div v-if="c.probes" class="ct-row">
          <span class="ct-label">Probes</span>
          <div class="ct-val">
            <div v-if="c.probes.liveness">Liveness: {{ c.probes.liveness }}</div>
            <div v-if="c.probes.readiness">Readiness: {{ c.probes.readiness }}</div>
            <div v-if="c.probes.startup">Startup: {{ c.probes.startup }}</div>
          </div>
        </div>
        <div v-if="c.securityContext" class="ct-row">
          <span class="ct-label">Security Context</span>
          <div class="ct-val"><div v-for="(v, k) in c.securityContext" :key="k">{{ k }}: {{ v }}</div></div>
        </div>
        <div v-if="c.terminationMessagePath" class="ct-row"><span class="ct-label">TerminationMessagePath</span><span class="ct-val">{{ c.terminationMessagePath }}</span></div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, inject } from 'vue'
import ContainerMetricsViewer from '../../metrics/ContainerMetricsViewer.vue'
import PortForwardControl from './PortForwardControl.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()
const clusterName = inject(clusterNameKey, ref(undefined))
const containers = computed(() => props.resource?.containers ?? [])
const collapsed = ref<Set<string>>(new Set())

function toggle(name: string) {
  const s = new Set(collapsed.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  collapsed.value = s
}
function statusClass(c: any): string {
  if (c.state === 'running' && c.ready) return 'ct-ok'
  if (c.state === 'running') return 'ct-warn'
  if (['CrashLoopBackOff','Error','OOMKilled'].includes(c.state)) return 'ct-err'
  return 'ct-neutral'
}
function fmtRes(r: Record<string,string>): string {
  return Object.entries(r).map(([k,v])=>`${k}: ${v}`).join(', ')
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
.ct-body { padding: 12px 14px; border-top: 1px solid #2a2b40; display: flex; flex-direction: column; gap: 8px; }
.ct-metrics { grid-column: 1 / -1; margin-bottom: 8px; }
.ct-row { display: grid; grid-template-columns: 140px 1fr; gap: 10px; align-items: baseline; font-size: 0.875rem; }
.ct-label { color: #868d8a; font-weight: 500; } .ct-val { color: #d4d4d4; word-break: break-word; }
.ct-mono { font-family: 'Courier New', monospace; font-size: 0.8rem; }
.ct-port { display: flex; align-items: center; gap: 8px; margin-bottom: 4px; } .ct-port-name { color: #868d8a; font-size: 0.8rem; }
.ct-env { display: flex; gap: 6px; margin-bottom: 2px; } .ct-env-name { color: #60a5fa; font-weight: 500; } .ct-env-sep { color: #6b6d8a; } .ct-env-value { color: #e5e7eb; word-break: break-all; }
.ct-mount { display: flex; gap: 8px; margin-bottom: 2px; } .ct-mount-name { color: #60a5fa; } .ct-mount-path { color: #d4d4d4; } .ct-mount-ro { color: #facc15; font-size: 0.75rem; }
</style>
