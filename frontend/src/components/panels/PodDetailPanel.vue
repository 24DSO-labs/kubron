<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>{{ pod.name }}</h3>
      <button class="btn-close" @click="emit('close')">✕</button>
    </div>

    <div class="tabs flex space-x-4 border-b border-gray-700 mb-4">
      <button v-for="tab in tabs" :key="tab.id" :class="['tab', { active: activeTab === tab.id }]"
        @click="activeTab = tab.id">
        {{ tab.label }}
      </button>
    </div>

    <!-- ═══════════ PROPERTIES ═══════════ -->
    <div v-if="activeTab === 'properties'" class="detail-section text-gray-300">
      <div class="meta-row">
        <span class="meta-label">Created:</span>
        {{ pod.age }} ago ({{ pod.created }})
      </div>
      <div class="meta-row">
        <span class="meta-label">Name:</span>
        {{ pod.name }}
      </div>
      <div class="meta-row">
        <span class="meta-label">Namespace:</span>
        {{ pod.namespace }}
      </div>

      <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
        <span class="meta-label">
          Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!labelsExpanded">{{ Object.keys(pod.labels || {}).length }} Labels</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(pod.labels || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="meta-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
        <span class="meta-label">
          Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!annotationsExpanded">{{ Object.keys(pod.annotations || {}).length }} Annotations</span>
          <div v-else class="labels-list">
            <span v-for="[key, value] in Object.entries(pod.annotations || {})" :key="key" class="label-pair">
              <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
            </span>
          </div>
        </div>
      </div>

      <div class="meta-row"><span class="meta-label">Controlled By:</span> {{ pod.controlledBy }}</div>
      <div class="meta-row">
        <span class="meta-label">Status:</span>
        <span class="status-badge" :class="statusClass(pod.status)">{{ pod.status }}</span>
      </div>
      <div class="meta-row"><span class="meta-label">Node:</span> {{ pod.node }}</div>
      <div class="meta-row"><span class="meta-label">Pod IP:</span> {{ pod.ip }}</div>
      <div class="meta-row"><span class="meta-label">Service Account:</span> {{ pod.serviceAccount }}</div>
      <div class="meta-row"><span class="meta-label">QoS Class:</span> {{ pod.qos }}</div>
      <div class="meta-row"><span class="meta-label">Conditions:</span> {{ formatConditions(pod.conditions) }}</div>
      <div class="meta-row"><span class="meta-label">Node Selector:</span> {{ formatMap(pod.nodeSelector) }}</div>

      <!-- Tolerations -->
      <div class="meta-row cursor-pointer" @click="tolerationsExpanded = !tolerationsExpanded">
        <span class="meta-label">
          Tolerations <span class="expand-icon" :class="{ expanded: tolerationsExpanded }">›</span>
        </span>
        <div class="meta-content">
          <span v-if="!tolerationsExpanded">{{ pod.tolerations.length }}</span>
          <table v-else class="tolerations-table">
            <thead>
              <tr>
                <th>Key</th>
                <th>Operator</th>
                <th>Value</th>
                <th>Effect</th>
                <th>Seconds</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="t in pod.tolerations" :key="JSON.stringify(t)">
                <td>{{ t.key || '' }}</td>
                <td>{{ t.operator || '' }}</td>
                <td>{{ t.value || '' }}</td>
                <td>{{ t.effect || '' }}</td>
                <td>{{ t.tolerationSeconds || '' }}</td>
              </tr>
            </tbody>
          </table>
        </div>
      </div>

      <!-- Pod AntiAffinities -->
      <div class="meta-row cursor-pointer" @click="affinityExpanded = !affinityExpanded">
        <span class="meta-label">
          Pod Anti Affinities
          <span class="expand-icon" :class="{ expanded: affinityExpanded }">›</span>
        </span>
        <span v-if="!affinityExpanded" class="meta-content">
          {{ affinityRuleCount }} {{ affinityRuleCount === 1 ? 'Rule' : 'Rules' }}
        </span>
      </div>
      <div v-if="affinityExpanded" class="affinity-editor-wrapper">
        <div class="affinity-editor">
          <!-- Gutter -->
          <div class="editor-gutter">
            <div v-for="line in visibleLines" :key="'g' + line.flatIdx" class="gutter-line">
              <span class="line-num">{{ line.lineNum }}</span>
              <span v-if="line.foldable" class="fold-icon" :class="{ collapsed: line.collapsed }"
                @click.stop="toggleFold(line.flatIdx)">›</span>
              <span v-else class="fold-spacer"></span>
            </div>
          </div>
          <div class="gutter-border"></div>
          <!-- Code area with indent guides -->
          <!-- Code area with indent guides -->
          <div class="editor-code">
            <div v-for="line in visibleLines" :key="'c' + line.flatIdx" class="code-line">
              <!-- Indent area with overlay guides -->
              <span class="indent-area" v-if="line.indent > 0">
                <span v-for="level in line.indent" :key="level" class="indent-slot">
                  <span v-if="line.activeGuides.includes(level - 1)" class="indent-guide-line"></span>
                  <span class="indent-space">&nbsp;&nbsp;</span>
                </span>
              </span>
              <!-- Content (without leading spaces) -->
              <span class="code-content" v-html="line.contentHtml"></span>
            </div>

          </div>
        </div>
      </div>

  <!-- Pod Volumes -->
      <div class="meta-row cursor-pointer" @click="volumesExpanded = !volumesExpanded">
        <span class="meta-label">
          Pod Volumes
          <span class="expand-icon" :class="{ expanded: volumesExpanded }">›</span>
        </span>
        <span v-if="!volumesExpanded" class="meta-content">
          {{ pod.volumes?.length || 0 }} {{ (pod.volumes?.length || 0) === 1 ? 'volume' : 'volumes' }}
        </span>
      </div>

      <div v-if="volumesExpanded" class="volumes-wrapper">
        <div v-if="!volumeGroups.length" class="volumes-empty">None</div>

        <div
          v-for="group in volumeGroups"
          :key="group.type"
          class="vol-group"
        >
          <!-- Group header -->
          <div
            class="vol-group-header cursor-pointer"
            @click="toggleVolumeGroup(group.type)"
          >
            <span class="vol-group-name">{{ group.type }}</span>
            <span
              class="expand-icon"
              :class="{ expanded: !collapsedVolumeGroups.has(group.type) }"
            >›</span>
            <span
              v-if="collapsedVolumeGroups.has(group.type)"
              class="vol-group-count"
            >
              {{ group.items.length }} {{ group.items.length === 1 ? 'volume' : 'volumes' }}
            </span>
          </div>

          <!-- Group table -->
          <div v-if="!collapsedVolumeGroups.has(group.type)" class="vol-group-body">
            <table class="vol-table">
              <thead>
                <tr>
                  <th>Name</th>
                  <th v-for="col in group.columns" :key="col">{{ col }}</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="vol in group.items" :key="vol.name">
                  <td>{{ vol.name }}</td>
                  <td v-for="col in group.columns" :key="col">{{ vol.details[col] || '' }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>
    </div>

    <!-- ═══════════ METRICS ═══════════ -->
    <div v-if="activeTab === 'metrics' && clusterName" class="detail-section">
      <PodMetricsPanel :cluster-name="clusterName" :namespace="pod.namespace" :pod-name="pod.name" />
    </div>

    <!-- ═══════════ CONTAINERS ═══════════ -->
    <div v-if="activeTab === 'containers' && pod.containers && pod.containers.length > 0" class="detail-section">
      <h4>Containers</h4>
      <div v-for="c in pod.containers" :key="c.name" class="container-item">
        <span class="container-name">{{ c.name }}</span>
        <span class="status-badge" :class="statusClass(c.state)">{{ c.state }}</span>
        <span class="container-image">{{ c.image }}</span>
      </div>
    </div>

    <!-- ═══════════ LOGS ═══════════ -->
    <div v-if="activeTab === 'logs' && pod.containers && pod.containers.length > 0" class="detail-section">
      <div class="logs-header">
        <h4>Logs</h4>
        <select :value="logContainer" @change="onLogContainerChange" class="log-select">
          <option v-for="c in pod.containers" :key="c.name" :value="c.name">{{ c.name }}</option>
        </select>
        <select :value="logLines" @change="onLogLinesChange" class="log-select">
          <option :value="50">50</option>
          <option :value="100">100</option>
          <option :value="500">500</option>
          <option :value="1000">1000</option>
        </select>
        <button @click="emit('loadLogs')" class="btn btn-sm">🔄</button>
      </div>
      <pre class="log-output">{{ logs || 'Нажмите 🔄 для загрузки' }}</pre>
    </div>

    <!-- ═══════════ EXEC ═══════════ -->
    <div v-if="activeTab === 'exec'" class="detail-section">
      <div class="exec-header">
        <h4>🖥 Exec</h4>
      </div>
      <div class="exec-input-row">
        <input :value="execCmd" @input="onExecCmdChange" @keyup.enter="emit('runExec')" placeholder="ls -la /tmp"
          class="exec-input" />
        <button @click="emit('runExec')" class="btn btn-sm">▶ Run</button>
      </div>
      <pre v-if="execOutput" class="exec-output">{{ execOutput }}</pre>
    </div>

    <!-- ═══════════ TERMINAL ═══════════ -->
    <div v-if="activeTab === 'terminal'" class="detail-section">
      <button class="btn btn-terminal" @click="emit('openTerminal')">🖥 Open Terminal</button>
    </div>

    <!-- ═══════════ PORT FORWARD ═══════════ -->
    <div v-if="activeTab === 'portforward'" class="detail-section pf-section">
      <h4>🌍 Port Forward</h4>
      <div class="pf-form">
        <label>Local:</label>
        <input type="number" :value="pfLocalPort" @input="onLocalPortChange" class="pf-input" />
        <label>Pod:</label>
        <input type="number" :value="pfPodPort" @input="onPodPortChange" class="pf-input" />
        <button @click="emit('startPF')" class="btn btn-sm">▶ Start</button>
      </div>
      <div v-if="portForwards.length > 0" class="pf-list">
        <div v-for="s in portForwards" :key="s.id" class="pf-item">
          <span class="pf-info">localhost:{{ s.localPort }} → {{ s.pod }}:{{ s.podPort }}</span>
          <span class="status-badge" :class="s.status === 'active' ? 'status-running' : 'status-pending'">
            {{ s.status }}
          </span>
          <button class="btn-icon" @click="emit('stopPF', s.id)">⏹</button>
        </div>
      </div>
    </div>

    <!-- ═══════════ ACTIONS ═══════════ -->
    <div v-if="activeTab === 'actions'" class="detail-actions">
      <button class="btn btn-danger" @click="emit('delete')">🗑 Удалить Pod</button>
    </div>
  </div>
</template>
<script lang="ts" setup>


import { ref, computed } from 'vue'
import type {
  PortForwardSession,
  PodAntiAffinity,
  ConditionInfo,
  VolumeInfo,
} from '../../types'
import PodMetricsPanel from '../metrics/PodMetricsPanel.vue'

const affinityRuleCount = computed(() => {
  const a = props.pod.podAntiAffinity as PodAntiAffinity | undefined
  if (!a) return 0
  let count = 0
  if (a.preferredDuringSchedulingIgnoredDuringExecution?.length) count++
  if (a.requiredDuringSchedulingIgnoredDuringExecution?.length) count++
  return count
})

const props = defineProps<{
  pod: any
  logs: string
  logContainer: string
  logLines: number
  execCmd: string
  execOutput: string
  statusClass: (s: string) => string
  pfLocalPort: number
  pfPodPort: number
  portForwards: PortForwardSession[]
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'loadLogs'): void
  (e: 'runExec'): void
  (e: 'delete'): void
  (e: 'update:logContainer', v: string): void
  (e: 'update:logLines', v: number): void
  (e: 'update:execCmd', v: string): void
  (e: 'update:pfLocalPort', v: number): void
  (e: 'update:pfPodPort', v: number): void
  (e: 'startPF'): void
  (e: 'stopPF', id: string): void
  (e: 'openTerminal'): void
}>()

const activeTab = ref('properties')
const tabs = [
  { id: 'properties', label: 'Properties' },
  { id: 'metrics', label: 'Metrics' },
  { id: 'containers', label: 'Containers' },
  { id: 'logs', label: 'Logs' },
  { id: 'exec', label: 'Exec' },
  { id: 'terminal', label: 'Terminal' },
  { id: 'portforward', label: 'Port Forward' },
  { id: 'actions', label: 'Actions' },
]

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const tolerationsExpanded = ref(false)
const affinityExpanded = ref(true)

function formatMap(o: Record<string, string> | undefined): string {
  if (!o || !Object.keys(o).length) return 'None'
  return Object.entries(o).map(([k, v]) => k + '=' + v).join('\n')
}

function formatConditions(c: ConditionInfo[]): string {
  if (!c || !c.length) return 'None'
  return c.map((x) => x.type).join(' ')
}


function esc(s: string): string {
  return s
    .replace(/&/g, '&amp;')
    .replace(/</g, '&lt;')
    .replace(/>/g, '&gt;')
}

// ═══════ YAML tree ═══════

interface YamlNode {
  raw: string
  indent: number
  children: YamlNode[]
}

function getIndent(line: string): number {
  const m = line.match(/^(\s*)/)
  return m ? Math.floor(m[1].length / 2) : 0
}

function highlightContent(raw: string): string {
  if (!raw.trim()) return '&nbsp;'
  const content = raw.trimStart()
  let result = ''
  let rest = content

  if (rest.startsWith('- ')) {
    result += '<span class="hl-dash">-&nbsp;</span>'
    rest = rest.slice(2)
  }

  const ci = rest.indexOf(': ')
  if (ci > 0) {
    result +=
      '<span class="hl-key">' + esc(rest.slice(0, ci)) + '</span>' +
      '<span class="hl-colon">:&nbsp;</span>' +
      '<span class="hl-val">' + esc(rest.slice(ci + 2)) + '</span>'
  } else if (rest.endsWith(':')) {
    result +=
      '<span class="hl-key">' + esc(rest.slice(0, -1)) + '</span>' +
      '<span class="hl-colon">:</span>'
  } else {
    result += '<span class="hl-val">' + esc(rest) + '</span>'
  }
  return result
}

function buildTree(lines: string[]): YamlNode[] {
  const nodes: YamlNode[] = lines.map((l) => ({
    raw: l,
    indent: getIndent(l),
    children: [],
  }))
  const roots: YamlNode[] = []
  const stack: YamlNode[] = []

  for (const node of nodes) {
    while (stack.length && stack[stack.length - 1].indent >= node.indent) {
      stack.pop()
    }
    if (!stack.length) {
      roots.push(node)
    } else {
      stack[stack.length - 1].children.push(node)
    }
    if (node.raw.trimEnd().endsWith(':')) {
      stack.push(node)
    }
  }
  return roots
}

// ═══════ Folding ═══════

const collapsedSet = ref<Set<number>>(new Set())

function toggleFold(flatIdx: number) {
  const s = new Set(collapsedSet.value)
  if (s.has(flatIdx)) s.delete(flatIdx)
  else s.add(flatIdx)
  collapsedSet.value = s
}

// ═══════ Flat lines ═══════

interface FlatLine {
  contentHtml: string
  lineNum: number
  foldable: boolean
  collapsed: boolean
  flatIdx: number
  indent: number
  activeGuides: number[]
}

function flattenNodes(
  nodes: YamlNode[],
  counter: { idx: number },
  collapsed: Set<number>,
): FlatLine[] {
  const out: FlatLine[] = []
  for (const node of nodes) {
    const idx = counter.idx++
    const foldable = node.children.length > 0
    const isColl = collapsed.has(idx)
    out.push({
      contentHtml: highlightContent(node.raw),
      lineNum: idx + 1,
      foldable,
      collapsed: isColl,
      flatIdx: idx,
      indent: node.indent,
      activeGuides: [],
    })
    if (!isColl && foldable) {
      out.push(...flattenNodes(node.children, counter, collapsed))
    }
  }
  return out
}

function computeActiveGuides(lines: FlatLine[]): void {
  for (let i = 0; i < lines.length; i++) {
    const guides: number[] = []
    const cur = lines[i].indent
    for (let level = 0; level < cur; level++) {
      for (let j = i + 1; j < lines.length; j++) {
        if (lines[j].indent <= level) {
          break
        }
        if (lines[j].indent > level) {
          guides.push(level)
          break
        }
      }
    }
    lines[i].activeGuides = guides
  }
}

// ═══════ Build YAML ═══════

function buildYaml(a: PodAntiAffinity | undefined): string[] {
  if (!a) return []
  const out: string[] = []
  const sp = (n: number) => '  '.repeat(n)

  const pref = a.preferredDuringSchedulingIgnoredDuringExecution
  if (pref && pref.length) {
    out.push('preferredDuringSchedulingIgnoredDuringExecution:')
    pref.forEach((t) => {
      out.push(sp(1) + '- weight: ' + t.weight)
      out.push(sp(2) + 'podAffinityTerm:')
      out.push(sp(3) + 'labelSelector:')
      out.push(sp(4) + 'matchLabels:')
      const ml = t.podAffinityTerm.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) {
        out.push(sp(5) + k + ': ' + v)
      }
      out.push(sp(3) + 'topologyKey: ' + t.podAffinityTerm.topologyKey)
    })
  }

  const req = a.requiredDuringSchedulingIgnoredDuringExecution
  if (req && req.length) {
    out.push('requiredDuringSchedulingIgnoredDuringExecution:')
    req.forEach((t) => {
      out.push(sp(1) + '- labelSelector:')
      out.push(sp(2) + 'matchLabels:')
      const ml = t.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) {
        out.push(sp(3) + k + ': ' + v)
      }
      out.push(sp(2) + 'topologyKey: ' + t.topologyKey)
    })
  }

  return out
}

// ═══════ Computed ═══════

const yamlTree = computed(() => {
  const raw = buildYaml(props.pod.podAntiAffinity)
  if (!raw.length) return []
  return buildTree(raw)
})

const visibleLines = computed<FlatLine[]>(() => {
  if (!yamlTree.value.length) {
    return [
      {
        contentHtml: '<span class="hl-val">None</span>',
        lineNum: 1,
        foldable: false,
        collapsed: false,
        flatIdx: 0,
        indent: 0,
        activeGuides: [],
      },
    ]
  }
  const flat = flattenNodes(yamlTree.value, { idx: 0 }, collapsedSet.value)
  computeActiveGuides(flat)
  return flat
})

// ═══════ Volumes grouped by type ═══════

const volumesExpanded = ref(false)

// ═══════ Volumes grouped by type ═══════

interface VolumeGroup {
  type: string
  items: VolumeInfo[]
  columns: string[]
}

const collapsedVolumeGroups = ref<Set<string>>(new Set())

function toggleVolumeGroup(type: string) {
  const s = new Set(collapsedVolumeGroups.value)
  if (s.has(type)) s.delete(type)
  else s.add(type)
  collapsedVolumeGroups.value = s
}

const volumeGroups = computed<VolumeGroup[]>(() => {
  const vols: VolumeInfo[] = props.pod.volumes
  if (!vols || !vols.length) return []

  const map = new Map<string, VolumeInfo[]>()
  for (const v of vols) {
    const t = v.type || 'Unknown'
    if (!map.has(t)) map.set(t, [])
    map.get(t)!.push(v)
  }

  const groups: VolumeGroup[] = []
  for (const [type, items] of map) {
    const colSet = new Set<string>()
    for (const item of items) {
      for (const key of Object.keys(item.details)) {
        colSet.add(key)
      }
    }
    groups.push({
      type: formatTypeName(type),
      items,
      columns: Array.from(colSet),
    })
  }

  return groups
})

function formatTypeName(type: string): string {
  // "EmptyDir" → "Empty Dir", "ConfigMap" → "Config Map"
  return type.replace(/([a-z])([A-Z])/g, '$1 $2')
}

// ═══════ Event handlers ═══════

function onLogContainerChange(e: Event) {
  emit('update:logContainer', (e.target as HTMLSelectElement).value)
}
function onLogLinesChange(e: Event) {
  emit('update:logLines', Number((e.target as HTMLSelectElement).value))
}
function onExecCmdChange(e: Event) {
  emit('update:execCmd', (e.target as HTMLInputElement).value)
}
function onLocalPortChange(e: Event) {
  emit('update:pfLocalPort', Number((e.target as HTMLInputElement).value))
}
function onPodPortChange(e: Event) {
  emit('update:pfPodPort', Number((e.target as HTMLInputElement).value))
}
</script>

<style scoped>
.tabs {
  overflow-x: auto;
}

.tab {
  padding: 8px 16px;
  background: transparent;
  border: none;
  border-bottom: 2px solid transparent;
  color: gray;
  cursor: pointer;
}

.tab.active {
  border-bottom: 2px solid #3b82f6;
  color: white;
}

.meta-row {
  display: grid;
  grid-template-columns: 150px 1fr;
  align-items: baseline;
  margin-bottom: 8px;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

pre {
  white-space: pre-wrap;
  word-break: break-word;
  background: #1e1e1e;
  padding: 8px;
  border-radius: 4px;
  font-size: 0.875rem;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  margin-left: 4px;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.labels-list {
  font-size: 0.875rem;
  margin: 0;
  padding: 0;
}

.meta-content {
  margin: 0;
  padding: 0;
  text-align: left;
}

.label-pair {
  display: block;
  overflow-wrap: break-word;
}

.label-key {
  color: #60a5fa;
  font-weight: 500;
}

.label-value {
  color: #e5e7eb;
}

.tolerations-table {
  width: 100%;
}

.tolerations-table th,
.tolerations-table td {
  text-align: left;
  padding: 4px 8px;
}

.tolerations-table th {
  font-weight: bold;
  border-bottom: 1px solid #333;
}

/* ═══════ Affinity Editor ═══════ */

.affinity-editor-wrapper {
  margin-top: 4px;
  margin-left: 150px;
  margin-bottom: 12px;
  text-align: left;
}

.affinity-editor {
  display: flex;
  background: #1a1b2e;
  border: 1px solid #2d2d3d;
  border-radius: 6px;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 13px;
  line-height: 22px;
  max-height: 340px;
  overflow: auto;
}

/* Gutter */

.editor-gutter {
  flex-shrink: 0;
  padding: 10px 0;
  min-width: 60px;
  user-select: none;
  background: #1a1b2e;
  position: sticky;
  left: 0;
  z-index: 2;
}

.gutter-line {
  display: flex;
  align-items: center;
  height: 22px;
  padding: 0 4px;
}

.line-num {
  color: #4a4b6a;
  font-variant-numeric: tabular-nums;
  min-width: 24px;
  text-align: right;
  margin-right: 2px;
}

.fold-icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  font-size: 12px;
  color: #6b6d8a;
  cursor: pointer;
  border-radius: 3px;
  transition: transform 0.15s, color 0.15s;
  transform: rotate(90deg);
  flex-shrink: 0;
}

.fold-icon:hover {
  color: #a0a2c0;
  background: rgba(255, 255, 255, 0.06);
}

.fold-icon.collapsed {
  transform: rotate(0deg);
  color: #8888aa;
}

.fold-spacer {
  display: inline-block;
  width: 16px;
  flex-shrink: 0;
}

.gutter-border {
  flex-shrink: 0;
  width: 1px;
  background: #2e2f4a;
  position: sticky;
  left: 60px;
  z-index: 2;
}

/* Code area */

.editor-code {
  flex: 1;
  padding: 10px 0;
  overflow-x: auto;
  min-width: 0;
}

.code-line {
  display: flex;
  align-items: stretch;
  height: 22px;
  line-height: 22px;
  white-space: nowrap;
}

/* Indent area */

.indent-area {
  display: inline-flex;
  flex-shrink: 0;
}

/* Each indent level slot: same width as 2 spaces in the monospace font */
.indent-slot {
  position: relative;
  display: inline-block;
  /* 2ch = width of two characters in monospace font */
  width: 2ch;
  height: 22px;
}

/* The two nbsp spaces that create the actual visual indent */
.indent-space {
  visibility: hidden;
}

/* Vertical guide line drawn inside the slot */
.indent-guide-line {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 1px;
  background: #2e2f4a;
  pointer-events: none;
}

/* Content */
.code-content {
  white-space: nowrap;
  padding-right: 12px;
}

/* Syntax highlighting */

.code-line :deep(.hl-key) {
  color: #00d4aa;
  font-weight: 500;
}

.code-line :deep(.hl-colon) {
  color: #6b6d8a;
}

.code-line :deep(.hl-val) {
  color: #4fc1c8;
}

.code-line :deep(.hl-dash) {
  color: #6b6d8a;
}

/* Scrollbar */

.affinity-editor::-webkit-scrollbar {
  width: 8px;
  height: 8px;
}

.affinity-editor::-webkit-scrollbar-track {
  background: #1a1b2e;
}

.affinity-editor::-webkit-scrollbar-thumb {
  background: #35365a;
  border-radius: 4px;
}

.affinity-editor::-webkit-scrollbar-thumb:hover {
  background: #45466a;
}

.affinity-editor::-webkit-scrollbar-corner {
  background: #1a1b2e;
}

/* ═══════ Pod Volumes ═══════ */

.volumes-wrapper {
  margin-left: 0;
  margin-bottom: 12px;
  grid-column: 1 / -1;
}
.volumes-empty {
  color: #6b6d8a;
  padding: 4px 0 4px 150px;
}

/* Volume group */

.vol-group {
  margin-bottom: 4px;
}
.vol-group-header {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 8px 0 4px 0;
  user-select: none;
  border-bottom: 1px solid #2a2b40;
}
.vol-group-name {
  font-weight: 500;
  color: #c0c2d8;
  font-size: 0.85rem;
}
.vol-group-count {
  color: #6b6d8a;
  font-size: 0.85rem;
  margin-left: 4px;
}

/* Volume table */

.vol-group-body {
  margin: 0;
}
.vol-table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
  table-layout: auto;
}
.vol-table th {
  text-align: left;
  padding: 6px 12px;
  color: #9ca0b8;
  font-weight: 700;
  font-size: 0.8rem;
  border-bottom: 1px solid #2a2b40;
  white-space: nowrap;
}
.vol-table td {
  text-align: left;
  padding: 6px 12px;
  color: #d4d4d4;
  border-bottom: 1px solid #1e1f33;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 250px;
}
.vol-table tr:last-child td {
  border-bottom: none;
}
</style>