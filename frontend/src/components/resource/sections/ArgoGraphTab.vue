<template>
  <div class="ag">
    <div class="ag-toolbar">
      <select v-model="kindFilter" class="ag-select" title="Kind">
        <option value="">All kinds</option>
        <option v-for="k in kinds" :key="k" :value="k">{{ k }}</option>
      </select>
      <label class="ag-check"><input v-model="onlyProblems" type="checkbox" /> Only unhealthy / out of sync</label>
      <button class="rd-btn" :disabled="loading" @click="load">Refresh</button>
      <button
        class="rd-btn"
        :title="expanded ? 'Restore panel width' : 'Expand the panel to the full content width'"
        :aria-pressed="expanded"
        @click="toggleExpanded"
      >{{ expanded ? 'Collapse' : 'Expand' }}</button>
      <span class="ag-legend">
        <span class="ag-leg ag-owns">owns</span>
        <span class="ag-leg ag-selects">selects</span>
        <span class="ag-leg ag-routes">routes</span>
      </span>
    </div>

    <div v-if="error" class="rd-empty">{{ error }}</div>
    <div v-else-if="loading && !graph" class="rd-empty">Loading graph…</div>
    <template v-else-if="graph">
      <div v-if="graph.truncated" class="ag-note ag-warn">Graph is limited to {{ graph.nodes.length }} nodes</div>
      <div v-for="w in graph.warnings" :key="w" class="ag-note ag-warn">{{ w }}</div>

      <div v-if="layout.nodes.length > 1" class="ag-scroll">
        <svg :width="layout.width + 2" :height="layout.height + 2" class="ag-svg" role="img" aria-label="Application resource graph">
          <path v-for="e in layout.edges" :key="e.id" :d="e.d" class="ag-edge" :class="'ag-edge-' + e.type" />
          <g
            v-for="n in layout.nodes"
            :key="n.id"
            class="ag-node"
            :class="[nodeTone(n), { 'ag-click': isClickable(n), 'ag-ghost': !n.managed && !n.isRoot }]"
            :transform="`translate(${n.x},${n.y})`"
            @click="onNodeClick(n)"
          >
            <title>{{ tooltip(n) }}</title>
            <rect :width="n.w" :height="n.h" rx="6" />
            <text x="10" y="19" class="ag-kind">{{ n.kind }}<tspan v-if="n.hook"> · hook</tspan></text>
            <text x="10" y="35" class="ag-name">{{ truncate(n.name, 26) }}</text>
            <text v-if="n.status === 'OutOfSync'" :x="n.w - 10" y="19" text-anchor="end" class="ag-oos">OutOfSync</text>
          </g>
        </svg>
      </div>
      <div v-else class="rd-empty">No resources reported by Argo CD</div>

      <div v-if="collapsedGroups.length" class="ag-note">
        Collapsed groups (click a group or a button to expand):
        <button v-for="g in collapsedGroups" :key="g.id" class="rd-btn ag-expand" @click="expand(g.id)">
          {{ g.name }}
        </button>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { getResourceById } from '../../../resources/registry'
import { filterGraph, layoutGraph, needsAttention, GROUP_PREFIX, type PlacedNode } from '../../../resources/argoGraph'
import { useDetailPanelSize } from '../../../composables/useDetailPanelSize'
import { argoLiveTick } from '../../../composables/useArgoLive'
import type { ArgoApplicationInfo, ArgoGraph } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const { expanded, toggleExpanded } = useDetailPanelSize()

const graph = ref<ArgoGraph | null>(null)
const loading = ref(false)
const error = ref('')
const kindFilter = ref('')
const onlyProblems = ref(false)
const expandedGroups = ref<Set<string>>(new Set())   // было: expanded


let seq = 0
async function load() {
  const ns = props.resource?.namespace
  const name = props.resource?.name
  if (!ns || !name) return
  const my = ++seq
  loading.value = true
  try {
    const g = (await callApp('GetArgoApplicationGraph', ns, name)) as ArgoGraph
    if (my !== seq) return // устаревший ответ (быстрая смена приложения или события)
    graph.value = g
    error.value = ''
  } catch (e: any) {
    if (my !== seq) return
    error.value = e?.message || String(e)
  } finally {
    if (my === seq) loading.value = false
  }
}

onMounted(load)
// Приложение другое: сбросить граф и фильтры; то же приложение изменилось (sync/health/revision, argocd:changed) — перечитать
watch(() => `${props.resource?.namespace}/${props.resource?.name}`, () => {
  graph.value = null
  expandedGroups.value = new Set()
  load()
})
watch(
  () => [props.resource?.syncStatus, props.resource?.healthStatus, props.resource?.revision, argoLiveTick.value],
  load,
)

const kinds = computed(() => [...new Set((graph.value?.nodes ?? []).map(n => n.kind))].sort())

const isProblem = needsAttention

const filtered = computed<ArgoGraph | null>(() => {
  const g = graph.value
  if (!g) return null
  if (!kindFilter.value && !onlyProblems.value) return g
  return filterGraph(g, n =>
    (!kindFilter.value || n.kind === kindFilter.value) && (!onlyProblems.value || isProblem(n)))
})

const layout = computed(() =>
  layoutGraph(filtered.value ?? { nodes: [], edges: [], truncated: false, warnings: [] }, props.resource?.name ?? '', expandedGroups.value))

const collapsedGroups = computed(() => layout.value.nodes.filter(n => n.id.startsWith(GROUP_PREFIX)))

function expand(id: string) {
  expandedGroups.value = new Set(expandedGroups.value).add(id)
}

function nodeTone(n: PlacedNode): string {
  if (n.isRoot) return 'ag-root'
  switch (n.health) {
    case 'Healthy': return 'ag-ok'
    case 'Degraded': case 'Missing': return 'ag-err'
    case 'Progressing': case 'Suspended': return 'ag-warn-n'
    default: return n.status === 'OutOfSync' ? 'ag-warn-n' : 'ag-unk'
  }
}

/** Открывается только то, что приложение умеет показывать; схлопнутая группа и корень не кликабельны */
function target(n: PlacedNode): { id: string; ns: string; name: string } | null {
  if (n.isRoot || n.id.startsWith(GROUP_PREFIX)) return null
  const def = getResourceById(n.kind)
  if (!def || !n.name) return null
  if (def.namespaced && !n.namespace) return null
  return { id: def.id, ns: def.namespaced ? n.namespace : '', name: n.name }
}
const isClickable = (n: PlacedNode) => n.id.startsWith(GROUP_PREFIX) || !!target(n)
function onNodeClick(n: PlacedNode) {
  if (n.id.startsWith(GROUP_PREFIX)) return expand(n.id)
  const t = target(n)
  if (t) emit('open', t)
}

const tooltip = (n: PlacedNode) =>
  [`${n.kind} ${n.name}`, n.namespace && `namespace: ${n.namespace}`, n.status && `sync: ${n.status}`,
    n.health && `health: ${n.health}`, n.message].filter(Boolean).join('\n')

const truncate = (s: string, max: number) => (s.length > max ? s.slice(0, max - 1) + '…' : s)
</script>

<style scoped>
.ag-toolbar { display: flex; align-items: center; gap: 12px; padding: 8px 0; flex-wrap: wrap; font-size: 0.85rem; }
.ag-select {
  padding: 5px 8px; background: #1a1b2e; color: #d4d4d4; border: 1px solid #3b3d5c; border-radius: 4px; font: inherit;
}
.ag-check { display: flex; align-items: center; gap: 6px; color: #d4d4d4; cursor: pointer; }
.ag-legend { margin-left: auto; display: flex; gap: 10px; font-size: 0.75rem; }
.ag-leg::before { content: ''; display: inline-block; width: 18px; height: 0; margin-right: 5px; vertical-align: middle; border-top: 2px solid; }
.ag-owns::before { border-color: #6b6d8a; }
.ag-selects::before { border-color: #60a5fa; border-top-style: dashed; }
.ag-routes::before { border-color: #a78bfa; }
.ag-note { padding: 6px 0; font-size: 0.8rem; color: #9ca3af; }
.ag-warn { color: #facc15; }
.ag-expand { margin-left: 6px; }
.ag-scroll { overflow: auto; max-height: calc(100vh - 340px); min-height: 240px; border: 1px solid #2a2f47; border-radius: 6px; padding: 10px; background: #12141f; }
.ag-svg { display: block; }

.ag-edge { fill: none; stroke: #6b6d8a; stroke-width: 1.4; }
.ag-edge-selects { stroke: #60a5fa; stroke-dasharray: 5 4; }
.ag-edge-routes { stroke: #a78bfa; }

.ag-node rect { fill: #1a1d2e; stroke: #3b3d5c; stroke-width: 1.4; }
.ag-ok rect { stroke: #4ade80; }
.ag-err rect { stroke: #f87171; fill: rgba(248, 113, 113, 0.08); }
.ag-warn-n rect { stroke: #fbbf24; }
.ag-unk rect { stroke: #6b6d8a; }
.ag-root rect { stroke: #8b5cf6; fill: rgba(139, 92, 246, 0.12); }
.ag-ghost rect { stroke-dasharray: 4 3; }
.ag-click { cursor: pointer; }
.ag-click:hover rect { fill: #20243a; }
.ag-kind { fill: #8b91ad; font-size: 10px; text-transform: uppercase; letter-spacing: 0.03em; }
.ag-name { fill: #e6e8f5; font-size: 12px; font-family: 'Courier New', monospace; }
.ag-oos { fill: #fbbf24; font-size: 9px; font-weight: 600; }
</style>
