<template>
  <div v-if="deployment" class="panel-wrapper">
    <K8sResourceDetailBase :resource="deployment" resource-icon="" :custom-tabs="customTabs" @close="$emit('close')"
      @edit="$emit('edit')" @delete="$emit('delete')">
      <template #actions>
        <!-- Scale -->
        <button class="btn-action" title="Scale deployment" @click="$emit('scale', deployment)">
          <img :src="scaleIcon" alt="Scale" class="btn-icon" />
        </button>

        <!-- Restart -->
        <button class="btn-action" title="Restart deployment"
          @click="$emit('restart', deployment.namespace, deployment.name)">
          <img :src="restartIcon" alt="Restart" class="btn-icon" />
        </button>

        <!-- Logs -->
        <button class="btn-action" title="View logs" @click="$emit('logs', deployment.namespace, deployment.name)">
          <img :src="logsIcon" alt="Logs" class="btn-icon" />
        </button>
      </template>

      <template #properties>
        <div class="dep-properties">
        <!-- Deployment Metrics (Collapsible) -->
        <div v-if="deployment.selector" class="metrics-section">
          <div class="metrics-header" @click="metricsExpanded = !metricsExpanded">
            <span class="metrics-header-title">Metrics</span>
            <span class="metrics-toggle" :class="{ collapsed: !metricsExpanded }"></span>
          </div>
          <div v-if="metricsExpanded" class="metrics-content">
            <DeploymentMetricsViewer
              :cluster-name="clusterName || ''"
              :namespace="deployment.namespace"
              :deployment-name="deployment.name"
              :selector="deployment.selector"
            />
          </div>
        </div>



        
          <!-- Created -->
          <div class="meta-row">
            <span class="meta-label">Created:</span>
            <span v-if="deployment.created" class="meta-value">
              {{ formatCreatedTime(deployment.created) }}
            </span>
            <span v-else class="meta-value" style="color: #6b6d8a;">—</span>
          </div>

          <!-- Labels -->
          <div class="meta-row cursor-pointer" @click="labelsExpanded = !labelsExpanded">
            <span class="meta-label">
              Labels <span class="expand-icon" :class="{ expanded: labelsExpanded }">›</span>
            </span>
            <div class="meta-content">
              <span v-if="!labelsExpanded" class="meta-summary">{{ Object.keys(deployment.labels || {}).length }}
                Labels</span>
              <div v-else class="labels-list">
                <span v-for="[key, value] in Object.entries(deployment.labels || {})" :key="key" class="label-pair">
                  <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
                </span>
              </div>
            </div>
          </div>

          <!-- Annotations -->
          <div class="meta-row cursor-pointer" @click="annotationsExpanded = !annotationsExpanded">
            <span class="meta-label">
              Annotations <span class="expand-icon" :class="{ expanded: annotationsExpanded }">›</span>
            </span>
            <div class="meta-content">
              <span v-if="!annotationsExpanded" class="meta-summary">{{ Object.keys(deployment.annotations || {}).length
                }} Annotations</span>
              <div v-else class="labels-list">
                <span v-for="[key, value] in Object.entries(deployment.annotations || {})" :key="key"
                  class="label-pair">
                  <span class="label-key">{{ key }}</span>=<span class="label-value">{{ value }}</span>
                </span>
              </div>
            </div>
          </div>

          <!-- Replicas -->
          <div class="meta-row">
            <span class="meta-label">Replicas:</span>
            {{ replicasSummary }}
          </div>

          <!-- Selector -->
          <div class="meta-row">
            <span class="meta-label">Selector:</span>
            <span v-if="deployment.selector" class="meta-value">{{ deployment.selector }}</span>
            <span v-else class="meta-value" style="color: #6b6d8a;">—</span>
          </div>

          <!-- Node Selector -->
          <div class="meta-row">
            <span class="meta-label">Node Selector:</span>
            <span v-if="deployment.nodeSelector" class="meta-value">{{ deployment.nodeSelector }}</span>
            <span v-else class="meta-value" style="color: #6b6d8a;">—</span>
          </div>

          <!-- Strategy Type -->
          <div class="meta-row">
            <span class="meta-label">Strategy Type:</span>
            {{ deployment.strategy }}
          </div>

          <!-- Status -->
          <div class="meta-row">
            <span class="meta-label">Status:</span>
            <span class="status-badge" :class="statusBadgeClass">{{ statusText }}</span>
          </div>

          <!-- Conditions -->
          <div v-if="deployment.conditions && deployment.conditions.length" class="meta-row">
            <span class="meta-label">Conditions</span>
            <div class="conditions-badges">
              <span v-for="cond in deployment.conditions" :key="cond.type" class="condition-badge"
                :class="'badge-' + cond.status.toLowerCase()">
                {{ cond.type }}
              </span>
            </div>
          </div>



          <!-- Pod Anti Affinities -->
          <div class="meta-row cursor-pointer" @click="affinityExpanded = !affinityExpanded">
            <span class="meta-label">
              Pod Anti Affinities <span class="expand-icon" :class="{ expanded: affinityExpanded }">›</span>
            </span>
            <span v-if="!affinityExpanded && deployment.podAntiAffinity" class="meta-content">
              {{ affinityRuleCount }} {{ affinityRuleCount === 1 ? 'Rule' : 'Rules' }}
            </span>
            <span v-else-if="!deployment.podAntiAffinity" class="meta-value" style="color: #6b6d8a;">—</span>
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

      <template #tab-revisions>
        <div class="revisions-section">
          <div v-if="sortedRevisions && sortedRevisions.length" class="revisions-container">
            <table class="revisions-table">
              <thead>
                <tr>
                  <th class="col-revision">#</th>
                  <th class="col-summary">Summary</th>
                  <th class="col-pods">Pods</th>
                  <th class="col-age">Age</th>
                  <th class="col-actions"></th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="(revision, idx) in sortedRevisions" :key="idx" class="revision-row">
                  <td class="col-revision">{{ revision.revision || idx + 1 }}</td>
                  <td class="col-summary">{{ revision.name }}</td>
                  <td class="col-pods">{{ revision.ready }}/{{ revision.replicas }}</td>
                  <td class="col-age">{{ revision.age }}</td>
                  <td class="col-actions">
                    <div class="action-menu-wrapper">
                      <button class="btn-menu"
                        @click="showRevisionMenu = showRevisionMenu === idx ? -1 : idx">⋮</button>
                      <div v-if="showRevisionMenu === idx" class="revision-menu">
                        <button class="menu-item" @click="rollbackToRevision(revision.revision)">Rollback</button>
                      </div>
                    </div>
                  </td>
                </tr>
              </tbody>
            </table>
          </div>
          <div v-else class="empty-state-text">No revisions available</div>
        </div>
      </template>

      <template #tab-pods>
        <div class="pods-section">
          <div v-if="deploymentPods && deploymentPods.length" class="pods-container">
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
                <tr v-for="pod in deploymentPods" :key="`${pod.namespace}-${pod.name}`" class="pod-row"
                  @click="selectPod(pod)">
                  <td class="col-name">{{ pod.name }}</td>
                  <td class="col-node">{{ pod.node || '—' }}</td>
                  <td class="col-namespace">{{ pod.namespace }}</td>
                  <td class="col-ready">{{ pod.ready }}</td>
                  <td class="col-cpu">{{ podMetricsMap[pod.name]?.totalCpu || '—' }}</td>
                  <td class="col-memory">{{ podMetricsMap[pod.name]?.totalMem || '—' }}</td>
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
import type { DeploymentInfo, PodAntiAffinityInfo, ConditionInfo, PodInfo } from '../../../types'
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import type { DetailTab } from '../K8sResourceDetailBase.vue'
import scaleIcon from '../../../assets/icons/scale.svg'
import restartIcon from '../../../assets/icons/restart.svg'
import logsIcon from '../../../assets/icons/logs.svg'
import DeploymentMetricsViewer from '../../metrics/DeploymentMetricsViewer.vue'
import { useK8s } from '../../../composables/useK8s'

const props = defineProps<{ deployment: any; clusterName?: string }>()
const emit = defineEmits<{
  (e: 'close'): void
  (e: 'scale', dep: any): void
  (e: 'restart', ns: string, name: string): void
  (e: 'logs', ns: string, name: string): void
  (e: 'edit'): void
  (e: 'delete'): void
  (e: 'pod-selected', pod: PodInfo): void
}>()

const { rollbackDeployment: rollbackDeploymentAPI, pods, podMetrics, loadRevisions, revisions } = useK8s()

const customTabs: DetailTab[] = [
  { id: 'revisions', label: 'Revisions' },
  { id: 'pods', label: 'Pods' }
]

const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const affinityExpanded = ref(false)
const metricsExpanded = ref(false)
const collapsedSet = ref<Set<number>>(new Set())
const showRevisionMenu = ref(-1)
const deploymentPods = ref<any[]>([])
const podMetricsMap = ref<Record<string, any>>({})
const revisionsLoaded = ref(false)
const metricsTabs = ['CPU', 'Memory', 'Network', 'Disk I/O']
const activeMetricTab = ref('CPU')

async function rollbackToRevision(revision: string): Promise<void> {
  try {
    if (!props.deployment) return
    const revisionNum = parseInt(revision, 10)
    if (!confirm(`Rollback ${props.deployment.name} to revision ${revision}?`)) return
    await rollbackDeploymentAPI(props.deployment.namespace, props.deployment.name, revisionNum)
    console.log('Rollback completed for revision:', revision)
    showRevisionMenu.value = -1
    window.dispatchEvent(new CustomEvent('deployment-rollback', { detail: { namespace: props.deployment.namespace, name: props.deployment.name } }))
  } catch (err) {
    console.error('Rollback failed:', err)
  }
}

function selectPod(pod: any): void {
  emit('pod-selected', pod)
}

async function loadDeploymentPods() {
  if (!props.deployment) {
    deploymentPods.value = []
    return
  }

  try {
    const { getPods } = useK8s()
    console.log(`Loading pods for namespace: ${props.deployment.namespace}`)
    const allPods = await getPods(props.deployment.namespace) as PodInfo[]

    console.log(`Got ${allPods?.length || 0} pods from GetPods`)

    const filtered = (allPods || []).filter((pod: PodInfo) => {
      if (pod.namespace !== props.deployment?.namespace) return false

      if (pod.controlledBy) {
        if (pod.controlledBy.includes(props.deployment.name)) {
          console.log(`Pod ${pod.name} matches via controlledBy: ${pod.controlledBy}`)
          return true
        }
      }

      if (pod.labels && props.deployment.selector) {
        try {
          const selectorParts = (props.deployment.selector || '').split(',').map((s: string) => s.trim())
          const matches = selectorParts.every((part: string) => {
            const [key, value] = part.split('=').map((s: string) => s.trim())
            return pod.labels && pod.labels[key] === value
          })
          if (matches) {
            console.log(`Pod ${pod.name} matches via selector`)
          }
          return matches
        } catch (e) {
          console.error('Error parsing selector:', e)
          return false
        }
      }

      return false
    })

    console.log(`Filtered ${filtered.length} pods for deployment ${props.deployment.name} out of ${(allPods || []).length}`)
    deploymentPods.value = filtered

    const metricsMap: Record<string, any> = {}
    if (podMetrics.value && podMetrics.value.length > 0) {
      podMetrics.value.forEach((metric: any) => {
        if (metric.namespace === props.deployment?.namespace) {
          metricsMap[metric.name] = metric
        }
      })
    }
    podMetricsMap.value = metricsMap
  } catch (err) {
    console.error('Error loading deployment pods:', err)
    deploymentPods.value = []
  }
}

watch(() => props.deployment, () => {
  loadDeploymentPods()
  revisionsLoaded.value = false
}, { immediate: true })

watch(() => pods.value, () => {
  loadDeploymentPods()
}, { deep: true })

function formatCreatedTime(created: string): string {
  if (!created) return '—'
  const date = new Date(created)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffSecs = Math.floor(diffMs / 1000)

  if (diffSecs < 60) return 'just now'
  const diffMins = Math.floor(diffSecs / 60)
  if (diffMins < 60) return `${diffMins}m${diffSecs % 60}s ago`
  const diffHours = Math.floor(diffMins / 60)
  if (diffHours < 24) return `${diffHours}h${diffMins % 60}m ago`
  const diffDays = Math.floor(diffHours / 24)

  return `${diffDays}d${diffHours % 24}h${diffMins % 60}m ago (${date.toLocaleString()})`
}

const replicasSummary = computed(() => {
  if (!props.deployment) return '—'
  return `${props.deployment.readyReplicas} desired, ${props.deployment.updatedReplicas} updated, ${props.deployment.replicas} total, ${props.deployment.readyReplicas} available, ${Math.max(0, props.deployment.replicas - props.deployment.readyReplicas)} unavailable`
})

const statusText = computed(() => {
  if (!props.deployment || !props.deployment.conditions || props.deployment.conditions.length === 0) {
    return 'Pending'
  }

  const availableCondition = props.deployment.conditions.find((c: ConditionInfo) => c.type === 'Available')
  const progressingCondition = props.deployment.conditions.find((c: ConditionInfo) => c.type === 'Progressing')

  if (availableCondition?.status === 'True' && progressingCondition?.status === 'True') {
    return 'Running'
  }

  if (progressingCondition?.status === 'False') {
    if (progressingCondition?.reason) {
      return progressingCondition.reason
    }
    return 'Failed'
  }

  return 'Pending'
})

const statusBadgeClass = computed(() => {
  if (!props.deployment || !props.deployment.conditions || props.deployment.conditions.length === 0) {
    return 'status-pending'
  }

  const availableCondition = props.deployment.conditions.find((c: ConditionInfo) => c.type === 'Available')
  const progressingCondition = props.deployment.conditions.find((c: ConditionInfo) => c.type === 'Progressing')

  if (availableCondition?.status === 'True' && progressingCondition?.status === 'True') {
    return 'status-running'
  }

  if (progressingCondition?.status === 'False') {
    return 'status-failed'
  }

  return 'status-pending'
})

const affinityRuleCount = computed(() => {
  if (!props.deployment || !props.deployment.podAntiAffinity) return 0
  const a = props.deployment.podAntiAffinity as PodAntiAffinityInfo | undefined
  if (!a) return 0
  let count = 0
  if (a.preferredDuringSchedulingIgnoredDuringExecution?.length) count++
  if (a.requiredDuringSchedulingIgnoredDuringExecution?.length) count++
  return count
})

function toggleFold(flatIdx: number): void {
  const s = new Set(collapsedSet.value)
  if (s.has(flatIdx)) s.delete(flatIdx)
  else s.add(flatIdx)
  collapsedSet.value = s
}

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

function buildYaml(a: PodAntiAffinityInfo | undefined): string[] {
  if (!a) return []
  const out: string[] = []
  const sp = (n: number) => '  '.repeat(n)

  const pref = a.preferredDuringSchedulingIgnoredDuringExecution
  if (pref && pref.length) {
    out.push('preferredDuringSchedulingIgnoredDuringExecution:')
    pref.forEach((t: any) => {
      out.push(sp(1) + '- weight: ' + t.weight)
      out.push(sp(2) + 'podAffinityTerm:')
      out.push(sp(3) + 'labelSelector:')
      out.push(sp(4) + 'matchLabels:')
      const ml = t.podAffinityTerm?.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) out.push(sp(5) + k + ': ' + v)
      out.push(sp(3) + 'topologyKey: ' + (t.podAffinityTerm?.topologyKey || ''))
    })
  }

  const req = a.requiredDuringSchedulingIgnoredDuringExecution
  if (req && req.length) {
    out.push('requiredDuringSchedulingIgnoredDuringExecution:')
    req.forEach((t: any) => {
      out.push(sp(1) + '- labelSelector:')
      out.push(sp(2) + 'matchLabels:')
      const ml = t.podAffinityTerm?.labelSelector?.matchLabels || {}
      for (const [k, v] of Object.entries(ml)) out.push(sp(3) + k + ': ' + v)
      out.push(sp(2) + 'topologyKey: ' + (t.podAffinityTerm?.topologyKey || ''))
    })
  }
  return out
}

const yamlTree = computed(() => {
  const raw = buildYaml(props.deployment?.podAntiAffinity)
  if (!raw.length) return []
  return buildTree(raw)
})

const sortedRevisions = computed(() => {
  if (!revisions.value || !revisions.value.length) return []
  return [...revisions.value].sort((a, b) => {
    const revA = parseInt(String(a.revision || 0), 10)
    const revB = parseInt(String(b.revision || 0), 10)
    return revB - revA
  })
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
</script>

<style scoped>
.dep-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  gap: 12px;
  font-size: 0.875rem;
  padding: 6px 0;
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}

.meta-label {
  font-weight: 600;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-size: 0.9rem;
  text-align: left;
}

.meta-value {
  color: #d4d4d4;
  text-align: left;
}

.meta-content {
  margin: 0;
  padding: 0;
  text-align: left;
}

.meta-summary {
  color: #6b6d8a;
  font-size: 0.85rem;
}

.expand-icon {
  display: inline-block;
  transition: transform 0.2s ease;
  margin-left: 4px;
}

.expand-icon.expanded {
  transform: rotate(90deg);
}

.label-list {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  font-size: 0.875rem;
  margin: 0;
  padding: 0;
}

.label-pair {
  display: block;
  overflow-wrap: break-word;
  margin: 2px 0;
}

.label-key {
  color: #60a5fa;
  font-weight: 500;
}

.label-value {
  color: #e5e7eb;
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

.conditions-badges {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.condition-badge {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 4px;
  font-size: 0.8rem;
  font-weight: 600;
}

.condition-badge.badge-true {

  color: #34d399;
}

.condition-badge.badge-false {

  color: #f87171;
}

.condition-badge.badge-unknown {

  color: #9ca3af;
}

/* ═══════ Affinity Editor ═══════ */

.affinity-editor-wrapper {
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

.cursor-pointer {
  cursor: pointer;
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

.metrics-placeholder {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100px;
  color: #6b6d8a;
  font-size: 0.9rem;
}

/* ═══════ Revisions Table ═══════ */

.revisions-section {
  padding: 0 !important;
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

.revisions-table thead tr {
  border: none;
}

.revisions-table th {
  padding: 10px 16px;
  text-align: left;
  font-weight: 600;
  color: #d4d4d4;
  border: none;
  white-space: nowrap;
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
}

.col-revision {
  width: 50px;
  min-width: 50px;
  text-align: center;
  font-weight: 600;
  color: #60a5fa;
}

.col-summary {
  flex: 1;
  min-width: 200px;
  word-break: break-all;
  color: #d4d4d4;
}

.col-pods {
  width: 80px;
  min-width: 80px;
  text-align: center;
  color: #d4d4d4;
  font-family: 'Courier New', monospace;
}

.col-age {
  width: 100px;
  min-width: 100px;
  text-align: right;
  color: #6b6d8a;
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

.revision-menu {
  position: absolute;
  top: 100%;
  right: 0;
  margin-top: 2px;
  background: #22233d;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 100;
  min-width: 120px;
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
  transition: background 0.15s;
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

.empty-state-text {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 40px 20px;
  color: #6b6d8a;
  font-size: 0.9rem;
}

/* ═══════ Pods Table ═══════ */

.pods-section {
  padding: 0 !important;
}

.pods-container {
  width: 100%;
  overflow-x: auto;
}

.pods-table {
  width: 100%;
  border-collapse: collapse;
  background: #1a1b2e;
  color: #d4d4d4;
  font-size: 0.9rem;
}

.pods-table thead {
  background: #1f2139;
  border-bottom: 1px solid #2a2b40;
  position: sticky;
  top: 0;
  z-index: 10;
}

.pods-table th {
  padding: 10px 16px;
  text-align: left;
  font-weight: 600;
  color: #d4d4d4;
  border: none;
  white-space: nowrap;
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

.btn-action.btn-danger {
  color: #ef4444;
}

.btn-action.btn-danger:hover {
  background: rgba(239, 68, 68, 0.2);
  border-color: #ef4444;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
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

.btn-action.btn-danger .btn-icon {
  filter: invert(1);
  opacity: 0.9;
}

.btn-action.btn-danger:hover .btn-icon {
  opacity: 1;
}
</style>
