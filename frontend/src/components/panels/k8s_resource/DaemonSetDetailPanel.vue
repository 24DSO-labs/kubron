<template>
  <div v-if="daemonset" class="panel-wrapper">
  <K8sResourceDetailBase :resource="daemonset" resource-icon="" :custom-tabs="customTabs" @close="$emit('close')"
      @edit="$emit('edit')" @delete="$emit('delete')" @namespaceClick="onNamespaceClick">
      <template #actions>
        <!-- Restart -->
        <button class="btn-action" title="Restart daemonset"
          @click="$emit('restart', daemonset.namespace, daemonset.name)">
          <img :src="restartIcon" alt="Restart" class="btn-icon" />
        </button>

        <!-- Logs -->
        <button class="btn-action" title="View logs" @click="$emit('logs', daemonset.namespace, daemonset.name)">
          <img :src="logsIcon" alt="Logs" class="btn-icon" />
        </button>
      </template>

      <template #properties>
        <div class="ds-properties">
          <!-- Metrics Section (Collapsible) -->
          <div v-if="daemonset.selector" class="metrics-section">
            <div class="metrics-header" @click="metricsExpanded = !metricsExpanded">
              <span class="metrics-header-title">Metrics</span>
              <span class="metrics-toggle" :class="{ collapsed: !metricsExpanded }"></span>
            </div>
            <div v-if="metricsExpanded" class="metrics-content">
              <DaemonSetMetricsViewer
                :cluster-name="clusterName || ''"
                :namespace="daemonset.namespace"
                :daemonset-name="daemonset.name"
                :selector="daemonset.selector"
              />
            </div>
          </div>

          <!-- Labels -->
          <div class="property-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
            <span class="property-label">
              Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
            </span>
            <div v-if="!labelsExpanded" class="property-value">{{ Object.keys(daemonset.labels || {}).length }} Labels</div>
            <div v-else class="property-details">
              <div v-for="[key, value] in Object.entries(daemonset.labels || {})" :key="key" class="detail-item">
                <span class="detail-key">{{ key }}</span>
                <span class="detail-value">{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- Annotations -->
          <div class="property-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
            <span class="property-label">
              Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
            </span>
            <div v-if="!annotationsExpanded" class="property-value">{{ Object.keys(daemonset.annotations || {}).length }} Annotations</div>
            <div v-else class="property-details">
              <div v-for="[key, value] in Object.entries(daemonset.annotations || {})" :key="key" class="detail-item">
                <span class="detail-key">{{ key }}</span>
                <span class="detail-value">{{ value }}</span>
              </div>
            </div>
          </div>

          <!-- Selector -->
          <div class="property-row">
            <span class="property-label">Selector</span>
            <span class="property-value">{{ daemonset.selector || '—' }}</span>
          </div>

          <!-- Node Selector -->
          <div class="property-row">
            <span class="property-label">Node Selector</span>
            <span class="property-value">{{ daemonset.nodeSelector || '—' }}</span>
          </div>

          <!-- Images -->
          <div class="property-row">
            <span class="property-label">Images</span>
            <div class="property-value images-list">
              <div v-for="(img, idx) in allContainerImages" :key="idx" class="image-item">{{ img }}</div>
              <span v-if="allContainerImages.length === 0">—</span>
            </div>
          </div>

          <!-- Strategy Type -->
          <div class="property-row">
            <span class="property-label">Strategy Type</span>
            <span class="property-value">{{ daemonset.strategy || '—' }}</span>
          </div>

          <!-- Tolerations -->
          <div class="property-row cursor-pointer" @click="tolerationsExpanded = !tolerationsExpanded">
            <span class="property-label">
              Tolerations <span class="expand-icon" :class="{ expanded: tolerationsExpanded }">›</span>
            </span>
            <span v-if="!tolerationsExpanded" class="property-value">{{ daemonset.tolerations ? daemonset.tolerations.length : 0 }}</span>
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
                <tr v-for="(tol, idx) in daemonset.tolerations" :key="idx">
                  <td>{{ tol.key || '' }}</td>
                  <td>{{ tol.operator || '' }}</td>
                  <td>{{ tol.value || '' }}</td>
                  <td>{{ tol.effect || '' }}</td>
                  <td>{{ tol.tolerationSeconds || '' }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <!-- Pod Status -->
          <div class="property-row">
            <span class="property-label">Pod Status</span>
            <span class="property-value">{{ podStatusSummary }}</span>
          </div>

          <!-- Node Affinities -->
          <div class="property-row cursor-pointer" @click="affinityExpanded = !affinityExpanded">
            <span class="property-label">
              Node Affinities <span class="expand-icon" :class="{ expanded: affinityExpanded }">›</span>
            </span>
          </div>
          <div v-if="affinityExpanded" class="affinity-editor-wrapper">
            <div class="affinity-editor">
              <div class="editor-gutter">
                <div v-for="line in visibleLines" :key="'g' + line.flatIdx" class="gutter-line">
                  <span class="line-num">{{ line.lineNum }}</span>
                  <span v-if="line.foldable" class="fold-icon" :class="{ collapsed: line.collapsed }"
                    @click.stop="toggleFold(line.flatIdx)">›</span>
                  <span v-else class="fold-spacer"></span>
                </div>
              </div>
              <div class="gutter-border"></div>
              <div class="editor-code">
                <div v-for="line in visibleLines" :key="'c' + line.flatIdx" class="code-line">
                  <span class="indent-area" v-if="line.indent > 0">
                    <span v-for="level in line.indent" :key="level" class="indent-slot">
                      <span v-if="line.activeGuides.includes(level - 1)" class="indent-guide-line"></span>
                      <span class="indent-space">&nbsp;&nbsp;</span>
                    </span>
                  </span>
                  <span class="code-content" v-html="line.contentHtml"></span>
                </div>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #tab-pods>
        <div class="pods-section">
          <div v-if="daemonsetPods && daemonsetPods.length" class="pods-container">
            <table class="pods-table">
              <thead>
                <tr>
                  <th class="col-name">Name</th>
                  <th class="col-node">Node</th>
                  <th class="col-namespace">Namespace</th>
                  <th class="col-ready">Ready</th>
                  <th class="col-cpu">CPU</th>
                  <th class="col-memory">Memory</th>
                  <th class="col-status">Status</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="pod in daemonsetPods" :key="`${pod.namespace}-${pod.name}`" class="pod-row"
                  @click="selectPod(pod)">
                  <td class="col-name">{{ pod.name }}</td>
                  <td class="col-node">{{ pod.node || '—' }}</td>
                  <td class="col-namespace">{{ pod.namespace }}</td>
                  <td class="col-ready">{{ pod.ready }}</td>
                  <td class="col-cpu">{{ formatCpu(podMetricsMap[pod.name]?.totalCpu) }}</td>
                  <td class="col-memory">{{ formatMemory(podMetricsMap[pod.name]?.totalMem) }}</td>
                  <td class="col-status">
                    <span class="status-badge" :class="'status-' + (pod.status || 'unknown').toLowerCase()">
                      {{ pod.status || 'Unknown' }}
                    </span>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="empty-state-text">No pods available</div>
        </div>
      </template>

      <template #events>
        <div class="empty-section">No events available</div>
      </template>
    </K8sResourceDetailBase>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import type { PodInfo, ConditionInfo } from '../../../types'
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import type { DetailTab } from '../K8sResourceDetailBase.vue'
import restartIcon from '../../../assets/icons/restart.svg'
import logsIcon from '../../../assets/icons/logs.svg'
import DaemonSetMetricsViewer from '../../metrics/DaemonSetMetricsViewer.vue'
import { useK8s } from '../../../composables/useK8s'

const props = defineProps<{ daemonset: any; clusterName?: string }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'pod-selected', pod: PodInfo): void
  (e: 'open-resource', data: { kind: string; name: string; namespace?: string }): void
}>()

const { pods, podMetrics } = useK8s()

const customTabs: DetailTab[] = [
  { id: 'pods', label: 'Pods' }
]

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const tolerationsExpanded = ref(false)
const affinityExpanded = ref(false)
const metricsExpanded = ref(false)
const collapsedSet = ref<Set<number>>(new Set())
const daemonsetPods = ref<any[]>([])
const podMetricsMap = ref<Record<string, any>>({})

interface YamlNode {
  raw: string
  indent: number
  children: YamlNode[]
}

interface FlatLine {
  contentHtml: string
  lineNum: number
  foldable: boolean
  collapsed: boolean
  flatIdx: number
  indent: number
  activeGuides: number[]
}

function getIndent(line: string): number {
  const m = line.match(/^(\s*)/)
  return m ? Math.floor(m[1].length / 2) : 0
}

function esc(s: string | any): string {
  return String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
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
    while (stack.length && stack[stack.length - 1].indent >= node.indent) stack.pop()
    if (!stack.length) roots.push(node)
    else stack[stack.length - 1].children.push(node)
    if (node.raw.trimEnd().endsWith(':')) stack.push(node)
  }
  return roots
}

function flattenNodes(nodes: YamlNode[], counter: { idx: number }, collapsed: Set<number>): FlatLine[] {
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
    if (!isColl && foldable) out.push(...flattenNodes(node.children, counter, collapsed))
  }
  return out
}

function computeActiveGuides(lines: FlatLine[]): void {
  for (let i = 0; i < lines.length; i++) {
    const guides: number[] = []
    const cur = lines[i].indent
    for (let level = 0; level < cur; level++) {
      for (let j = i + 1; j < lines.length; j++) {
        if (lines[j].indent <= level) break
        if (lines[j].indent > level) { guides.push(level); break }
      }
    }
    lines[i].activeGuides = guides
  }
}

function toggleFold(flatIdx: number): void {
  const s = new Set(collapsedSet.value)
  if (s.has(flatIdx)) s.delete(flatIdx)
  else s.add(flatIdx)
  collapsedSet.value = s
}

const podStatusSummary = computed(() => {
  if (!props.daemonset) return '—'
  const ready = props.daemonset.numberReady || 0
  return `Running: ${ready}`
})

const allContainerImages = computed(() => {
  const images = new Set<string>()
  if (daemonsetPods.value && daemonsetPods.value.length > 0) {
    daemonsetPods.value.forEach((pod: any) => {
      if (pod.containers && Array.isArray(pod.containers)) {
        pod.containers.forEach((container: any) => {
          if (container.image) {
            images.add(container.image)
          }
        })
      }
    })
  }
  return Array.from(images).sort()
})

function buildYaml(affinity: any): string[] {
  if (!affinity || !affinity.nodeAffinity) return []
  
  const out: string[] = []
  const sp = (n: number) => '  '.repeat(n)
  const na = affinity.nodeAffinity
  
  // Handle required during scheduling
  if (na.requiredDuringSchedulingIgnoredDuringExecution?.nodeSelectorTerms) {
    out.push('requiredDuringSchedulingIgnoredDuringExecution:')
    out.push(sp(1) + 'nodeSelectorTerms:')
    
    na.requiredDuringSchedulingIgnoredDuringExecution.nodeSelectorTerms.forEach((term: any) => {
      if (term.matchExpressions && term.matchExpressions.length > 0) {
        out.push(sp(2) + '- matchExpressions:')
        term.matchExpressions.forEach((expr: any) => {
          out.push(sp(3) + '- key: ' + expr.key)
          out.push(sp(4) + 'operator: ' + expr.operator)
          if (expr.values && expr.values.length > 0) {
            out.push(sp(4) + 'values:')
            expr.values.forEach((val: string) => {
              out.push(sp(5) + '- ' + val)
            })
          }
        })
      }
    })
  }
  
  // Handle preferred during scheduling
  if (na.preferredDuringSchedulingIgnoredDuringExecution && na.preferredDuringSchedulingIgnoredDuringExecution.length > 0) {
    if (out.length > 0) out.push('')
    out.push('preferredDuringSchedulingIgnoredDuringExecution:')
    
    na.preferredDuringSchedulingIgnoredDuringExecution.forEach((item: any) => {
      out.push(sp(1) + '- weight: ' + item.weight)
      out.push(sp(2) + 'preference:')
      if (item.preference?.matchExpressions && item.preference.matchExpressions.length > 0) {
        out.push(sp(3) + 'matchExpressions:')
        item.preference.matchExpressions.forEach((expr: any) => {
          out.push(sp(4) + '- key: ' + expr.key)
          out.push(sp(5) + 'operator: ' + expr.operator)
          if (expr.values && expr.values.length > 0) {
            out.push(sp(5) + 'values:')
            expr.values.forEach((val: string) => {
              out.push(sp(6) + '- ' + val)
            })
          }
        })
      }
    })
  }
  
  return out
}

const yamlTree = computed(() => {
  const raw = buildYaml(props.daemonset?.affinity)
  if (!raw.length) return []
  return buildTree(raw)
})

const visibleLines = computed<FlatLine[]>(() => {
  if (!yamlTree.value.length) {
    return [{
      contentHtml: '<span class="hl-val">None</span>',
      lineNum: 1, foldable: false, collapsed: false,
      flatIdx: 0, indent: 0, activeGuides: [],
    }]
  }
  const flat = flattenNodes(yamlTree.value, { idx: 0 }, collapsedSet.value)
  computeActiveGuides(flat)
  return flat
})

function openResource(kind: string, name: string) {
  console.log('DaemonSetDetailPanel.openResource called:', { kind, name, namespace: props.daemonset?.namespace })
  const data = {
    kind,
    name,
    namespace: props.daemonset?.namespace
  }
  console.log('Emitting open-resource:', data)
  emit('open-resource', data)
}

function onNamespaceClick(namespace: string) {
  console.log('DaemonSetDetailPanel.onNamespaceClick called with:', namespace)
  openResource('Namespace', namespace)
}

function selectPod(pod: any): void {
  emit('pod-selected', pod)
}

function formatCpu(value: string | number | undefined): string {
  if (!value) return '—'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '—'
  return num.toFixed(3)
}

function formatMemory(value: string | number | undefined): string {
  if (!value) return '—'
  const num = typeof value === 'string' ? parseFloat(value) : value
  if (isNaN(num)) return '—'
  const mib = num / (1024 * 1024)
  return mib.toFixed(2) + ' MiB'
}

async function loadDaemonSetPods() {
  if (!props.daemonset) {
    daemonsetPods.value = []
    return
  }

  try {
    const { getPods } = useK8s()
    const allPods = await getPods(props.daemonset.namespace) as PodInfo[]

    const filtered = (allPods || []).filter((pod: PodInfo) => {
      if (pod.namespace !== props.daemonset?.namespace) return false

      if (pod.controlledBy) {
        if (pod.controlledBy.includes(props.daemonset.name)) {
          return true
        }
      }

      if (pod.labels && props.daemonset.selector) {
        try {
          const selectorParts = (props.daemonset.selector || '').split(',').map((s: string) => s.trim())
          const matches = selectorParts.every((part: string) => {
            const [key, value] = part.split('=').map((s: string) => s.trim())
            return pod.labels && pod.labels[key] === value
          })
          return matches
        } catch (e) {
          console.error('Error parsing selector:', e)
          return false
        }
      }

      return false
    })

    daemonsetPods.value = filtered

    const metricsMap: Record<string, any> = {}
    if (podMetrics.value && podMetrics.value.length > 0) {
      podMetrics.value.forEach((metric: any) => {
        if (metric.namespace === props.daemonset?.namespace) {
          metricsMap[metric.name] = metric
        }
      })
    }

    podMetricsMap.value = metricsMap
  } catch (err) {
    console.error('Error loading daemonset pods:', err)
    daemonsetPods.value = []
  }
}

watch(() => props.daemonset, () => {
  loadDaemonSetPods()
}, { immediate: true, deep: true })

watch(() => pods.value, () => {
  loadDaemonSetPods()
}, { deep: true })
</script>

<style scoped>
.ds-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

/* ═══════ Metrics Section ═══════ */

.metrics-section {
  width: 100%;
  margin-bottom: 16px;
  border: 1px solid #2a2b40;
  border-radius: 6px;
  overflow: hidden;
  background: #1a1b2e;
}

.metrics-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.1) 0%, rgba(59, 130, 246, 0.05) 100%);
  border-bottom: 1px solid #2a2b40;
  cursor: pointer;
  user-select: none;
  transition: background 0.2s ease;
}

.metrics-header:hover {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0.1) 100%);
}

.metrics-header-title {
  flex: 1;
  font-weight: 600;
  color: #d4d4d4;
  font-size: 0.95rem;
  letter-spacing: 0.3px;
}

.metrics-toggle {
  display: inline-block;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  position: relative;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.metrics-toggle::before {
  content: '';
  position: absolute;
  top: 50%;
  left: 50%;
  width: 5px;
  height: 9px;
  border-right: 2px solid #6b6d8a;
  border-bottom: 2px solid #6b6d8a;
  transform: translate(-50%, -50%) rotate(-45deg);
  transition: border-color 0.2s ease;
}

.metrics-header:hover .metrics-toggle::before {
  border-color: #a0a2c0;
}

.metrics-toggle.collapsed {
  transform: rotate(-90deg);
}

.metrics-content {
  padding: 16px;
  max-height: 600px;
  overflow-y: auto;
}

.property-row {
  display: grid;
  grid-template-columns: 150px 1fr;
  gap: 16px;
  padding: 12px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  align-items: flex-start;
  font-size: 0.875rem;
}

.property-row:last-child {
  border-bottom: none;
}

.property-label {
  font-weight: 600;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.9rem;
  text-align: left;
  display: flex;
  align-items: center;
  gap: 6px;
}

.property-value {
  color: #d4d4d4;
  text-align: left;
  word-break: break-word;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  font-size: 1.2em;
  line-height: 1;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.property-details {
  grid-column: 1 / -1;
  padding: 12px 16px;
  background: rgba(255, 255, 255, 0.02);
  border-radius: 4px;
  border-left: 3px solid #60a5fa;
  margin-top: -8px;
  margin-left: 0;
}

.detail-item {
  display: grid;
  grid-template-columns: 150px 1fr;
  gap: 12px;
  padding: 6px 0;
  font-size: 0.85rem;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.detail-item:last-child {
  border-bottom: none;
}

.detail-key {
  color: #60a5fa;
  font-weight: 500;
  word-break: break-all;
}

.detail-value {
  color: #e5e7eb;
  word-break: break-all;
}

.images-list {
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.image-item {
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
  font-size: 0.85rem;
}

.status-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.85rem;
  font-weight: 600;
}

.status-running {
  color: #34d399;
}

.status-pending {
  color: #fbbf24;
}

.status-failed {
  color: #f87171;
}

.status-unknown {
  color: #9ca3af;
}

/* ═══════ Affinity Editor ═══════ */

.affinity-editor-wrapper {
  grid-column: 1 / -1;
  margin-top: 8px;
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

.indent-area {
  display: inline-flex;
  flex-shrink: 0;
}

.indent-slot {
  position: relative;
  display: inline-block;
  width: 2ch;
  height: 22px;
}

.indent-space {
  visibility: hidden;
}

.indent-guide-line {
  position: absolute;
  left: 0;
  top: 0;
  bottom: 0;
  width: 1px;
  background: #2e2f4a;
  pointer-events: none;
}

.code-content {
  white-space: nowrap;
  padding-right: 12px;
}

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

/* ═══════ Pods Table ═══════ */

.pods-section {
  padding: 0 !important;
}

.pods-container {
  width: 100%;
  overflow-x: auto !important;
  scrollbar-color: #35365a #1a1b2e !important;
  scrollbar-width: thin !important;
}

.pods-container::-webkit-scrollbar {
  width: 8px !important;
  height: 8px !important;
}

.pods-container::-webkit-scrollbar-track {
  background: #1a1b2e !important;
}

.pods-container::-webkit-scrollbar-thumb {
  background: #35365a !important;
  border-radius: 4px !important;
}

.pods-container::-webkit-scrollbar-thumb:hover {
  background: #45466a !important;
}

.pods-container::-webkit-scrollbar-corner {
  background: #1a1b2e !important;
}

.pods-table {
  width: 100%;
  border-collapse: collapse;
  background: #1a1b2e;
  color: #d4d4d4;
  font-size: 0.9rem;
}

.pods-table thead {
  background: rgba(255, 255, 255, 0.02);
  border-bottom: 1px solid #2a2b40;
  position: sticky;
  top: 0;
  z-index: 10;
}

.pods-table th {
  padding: 10px 16px !important;
  text-align: left !important;
  font-weight: 600 !important;
  border: none !important;
  white-space: nowrap !important;
  text-transform: uppercase !important;
  font-size: 0.75rem !important;
  letter-spacing: 0.5px !important;
  color: #868d8a !important;
  background: rgba(255, 255, 255, 0.02) !important;
}

.pods-table tbody tr {
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
  transition: background 0.15s;
  cursor: pointer;
}

.pods-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}

.pods-table td {
  padding: 12px 16px;
  border: none;
  text-align: left;
}

.pod-row {
  cursor: pointer;
}

.col-name {
  flex: 1;
  min-width: 120px;
  color: #60a5fa;
}

.col-node {
  width: 100px;
  min-width: 100px;
  color: #d4d4d4;
}

.col-namespace {
  width: 80px;
  min-width: 80px;
  color: #6b6d8a;
}

.col-ready {
  width: 50px;
  min-width: 50px;
  text-align: center;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-cpu {
  width: 60px;
  min-width: 60px;
  text-align: right;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-memory {
  width: 80px;
  min-width: 80px;
  text-align: right;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-status {
  width: 80px;
  min-width: 80px;
  text-align: left;
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

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.empty-state-text {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #6b6d8a;
  font-size: 0.9rem;
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

.cursor-pointer {
  cursor: pointer;
}

.tolerations-table {
  grid-column: 1 / -1;
  overflow-x: auto;
  margin-top: -8px;
  width: 100%;
}

.tolerations-table table {
  width: 100%;
  border-collapse: collapse;
  font-size: 0.85rem;
}

.tolerations-table thead {
  background: rgba(255, 255, 255, 0.05);
  border-bottom: 1px solid rgba(255, 255, 255, 0.1);
}

.tolerations-table th {
  padding: 8px 12px;
  text-align: left;
  font-weight: 600;
  color: #868d8a;
  border: none;
  white-space: nowrap;
}

.tolerations-table tbody tr {
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
  background: rgba(255, 255, 255, 0.01);
}

.tolerations-table tbody tr:hover {
  background: rgba(255, 255, 255, 0.03);
}

.tolerations-table td {
  padding: 10px 12px;
  color: #d4d4d4;
  border: none;
}
</style>
