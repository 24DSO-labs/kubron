<template>
  <K8sResourceDetailBase
    :resource="pod"
    resource-icon=""
    :custom-tabs="customTabs"
    @close="emit('close')"
    @edit="emit('edit')"
    @delete="emit('delete')"
    @namespace-click="openResource('Namespace', $event)"
  >
  >
    <!-- Two-level action menu -->
    <template #actions>
      <!-- Attach -->
      <div class="action-group">
        <button class="btn-action" title="Attach to container" @click="showAttachMenu = !showAttachMenu">
          <img :src="attachIcon" alt="Attach" class="btn-icon"/>
        </button>
        <div v-if="showAttachMenu && pod.containers.length > 0" class="action-dropdown">
          <div v-for="c in pod.containers" :key="c.name" class="dropdown-item" @click="handleAttach(c.name)">
            {{ c.name }}
          </div>
        </div>
        <div v-else-if="showAttachMenu" class="action-dropdown empty">No containers</div>
      </div>

      <!-- Terminal -->
      <div class="action-group">
        <button class="btn-action" title="Open terminal" @click="showTerminalMenu = !showTerminalMenu">
          <img :src="terminalIcon" alt="Terminal" class="btn-icon"/>
        </button>
        <div v-if="showTerminalMenu && pod.containers.length > 0" class="action-dropdown">
          <div v-for="c in pod.containers" :key="c.name" class="dropdown-item" @click="handleTerminal(c.name)">
            {{ c.name }}
          </div>
        </div>
        <div v-else-if="showTerminalMenu" class="action-dropdown empty">No containers</div>
      </div>

      <!-- Logs -->
      <div class="action-group">
        <button class="btn-action" title="View logs" @click="showLogsMenu = !showLogsMenu">
          <img :src="logsIcon" alt="Logs" class="btn-icon"/>
        </button>
        <div v-if="showLogsMenu && pod.containers.length > 0" class="action-dropdown">
          <div v-for="c in pod.containers" :key="c.name" class="dropdown-item" @click="handleLogs(c.name)">
            {{ c.name }}
          </div>
        </div>
        <div v-else-if="showLogsMenu" class="action-dropdown empty">No containers</div>
      </div>

      <!-- Evict -->
      <button class="btn-action" title="Evict pod" @click="emit('evict')">
        <img :src="evictIcon" alt="Evict" class="btn-icon"/>
      </button>
    </template>

    <!-- Properties tab content -->
    <template #properties>
      <div class="pod-properties">
        <!-- Pod Metrics (Collapsible) -->
        <div v-if="clusterName" class="metrics-section">
          <div class="metrics-header" @click="metricsExpanded = !metricsExpanded">
            <span class="metrics-header-title">Metrics</span>
            <span class="metrics-toggle" :class="{ collapsed: !metricsExpanded }"></span>
          </div>
          <div v-if="metricsExpanded" class="metrics-content">
            <PodMetricsViewer :cluster-name="clusterName" :namespace="pod.namespace" :pod-name="pod.name" />
          </div>
        </div>

        <!-- Labels (expandable) -->
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

        <!-- Annotations (expandable) -->
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

        <!-- Status and other info -->
        <div class="meta-row">
          <span class="meta-label">Controlled By:</span>
          <span v-if="pod.controlledBy" class="controlled-by-wrapper">
            <span class="controlled-by-type">{{ getControlledByType() }}</span>
            <a class="meta-link" @click="openControlledByResource()">
              {{ getControlledByName() }}
            </a>
          </span>
          <span v-else>—</span>
        </div>
        <div class="meta-row">
          <span class="meta-label">Status:</span>
          <span :class="['status-text', statusClass(pod.status)]">{{ pod.status }}</span>
        </div>
        <div class="meta-row">
          <span class="meta-label">Node:</span>
          <a v-if="pod.node" class="meta-link" @click="openResource('Node', pod.node)">
            {{ pod.node }}
          </a>
          <span v-else>—</span>
        </div>
        <div class="meta-row"><span class="meta-label">Pod IP:</span> {{ pod.ip }}</div>
        <div class="meta-row">
          <span class="meta-label">Service Account:</span>
          <a v-if="pod.serviceAccount" class="meta-link" @click="openResource('ServiceAccount', pod.serviceAccount, pod.namespace)">
            {{ pod.serviceAccount }}
          </a>
          <span v-else>—</span>
        </div>
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

        <!-- Pod Anti Affinities -->
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
          <div v-for="group in volumeGroups" :key="group.type" class="vol-group">
            <div class="vol-group-header cursor-pointer" @click="toggleVolumeGroup(group.type)">
              <span class="vol-group-name">{{ group.type }}</span>
              <span class="expand-icon" :class="{ expanded: !collapsedVolumeGroups.has(group.type) }">›</span>
              <span v-if="collapsedVolumeGroups.has(group.type)" class="vol-group-count">
                {{ group.items.length }} {{ group.items.length === 1 ? 'volume' : 'volumes' }}
              </span>
            </div>
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
    </template>

    <!-- Containers tab content -->
    <template #tab-containers v-if="pod.containers && pod.containers.length > 0">
      <div class="containers-section">
        <div v-for="c in pod.containers" :key="c.name" class="ct-block">
          <div class="ct-header cursor-pointer" @click="toggleContainer(c.name)">
            <div class="ct-header-left">
              <span class="ct-chevron" :class="{ expanded: !collapsedContainers.has(c.name) }">›</span>
              <span class="ct-name">{{ c.name }}</span>
              <span v-if="c.isInit" class="ct-badge-init">init</span>
            </div>
          </div>

          <div v-if="!collapsedContainers.has(c.name)" class="ct-body">
            <div v-if="clusterName && !c.isInit" class="ct-metrics-wrapper">
              <div class="ct-metrics-header" @click.stop="toggleContainerMetrics(c.name)">
                <span class="ct-metrics-title">Metrics</span>
                <span class="ct-metrics-toggle" :class="{ collapsed: collapsedContainerMetrics.has(c.name) }"></span>
              </div>
              <div v-if="!collapsedContainerMetrics.has(c.name)" class="ct-metrics-content">
                <ContainerMetricsViewer :cluster-name="clusterName" :namespace="pod.namespace" :pod-name="pod.name" :container-name="c.name" />
              </div>
            </div>

            <div class="ct-row">
              <span class="ct-label">Status</span>
              <span class="ct-val">
                <span :class="containerStatusColor(c)">{{ c.state }}{{ c.ready ? ', ready' : '' }}</span>
              </span>
            </div>

            <div class="ct-row ct-row-alt">
              <span class="ct-label">Image</span>
              <span class="ct-val">{{ c.image }}</span>
            </div>

            <div class="ct-row">
              <span class="ct-label">ImagePullPolicy</span>
              <span class="ct-val">{{ c.imagePullPolicy || '—' }}</span>
            </div>

           <div class="ct-row ct-row-alt ct-row-click" @click.stop="toggleContainerSection(c.name, 'ports')">
              <span class="ct-label">
                Ports
                <span class="ct-sec-toggle">{{ isContainerSectionCollapsed(c.name, 'ports') ? '›' : '×' }}</span>
              </span>
            </div>
            <div v-if="!isContainerSectionCollapsed(c.name, 'ports')" class="ct-row ct-sub-row ct-row-alt">
              <span class="ct-label"></span>
              <span class="ct-val">
                <PodPortsSection :namespace="pod.namespace" :pod-name="pod.name" :ports="c.ports"
                  :port-forwards="portForwards" :on-start-port-forward="onStartPortForward"
                  :on-stop-port-forward="onStopPortForward" />
              </span>
            </div>

            <div class="ct-row ct-row-click" @click.stop="toggleContainerSection(c.name, 'env')">
              <span class="ct-label">
                Environment
                <span class="ct-sec-toggle">{{ isContainerSectionCollapsed(c.name, 'env') ? '›' : '×' }}</span>
              </span>
              <span class="ct-val" v-if="isContainerSectionCollapsed(c.name, 'env')">
                {{ (c.env?.length || 0) }} {{ (c.env?.length || 0) === 1 ? 'Var' : 'Vars' }}
              </span>
              <span class="ct-val" v-else-if="!c.env || !c.env.length">None</span>
            </div>
            <template v-if="!isContainerSectionCollapsed(c.name, 'env') && c.env && c.env.length">
              <div v-for="(e, ei) in c.env" :key="e.name"
                :class="['ct-row', 'ct-sub-row', { 'ct-row-alt': (ei as number) % 2 === 0 }]">
                <span class="ct-label"></span>
                <span class="ct-val ct-env-cell">
                  <span class="ct-env-name">{{ e.name }}</span>
                  <span class="ct-env-sep">:</span>
                  <span class="ct-env-value">{{ e.value || '' }}</span>
                </span>
              </div>
            </template>

            <div class="ct-row ct-row-alt ct-row-click" @click.stop="toggleContainerSection(c.name, 'mounts')">
              <span class="ct-label">
                Mounts
                <span class="ct-sec-toggle">{{ isContainerSectionCollapsed(c.name, 'mounts') ? '›' : '×' }}</span>
              </span>
              <span class="ct-val" v-if="isContainerSectionCollapsed(c.name, 'mounts')">
                {{ (c.volumeMounts?.length || 0) }} {{ (c.volumeMounts?.length || 0) === 1 ? 'Mount' : 'Mounts' }}
              </span>
              <span class="ct-val" v-else-if="!c.volumeMounts || !c.volumeMounts.length">None</span>
            </div>

            <template v-if="!isContainerSectionCollapsed(c.name, 'mounts') && c.volumeMounts && c.volumeMounts.length">
              <template v-for="(m, mi) in c.volumeMounts" :key="m.name + ':' + m.mountPath">
                <div :class="['ct-row', 'ct-sub-row', { 'ct-row-alt': (mi as number) % 2 === 0 }]">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-mount-path">{{ m.mountPath }}</span>
                </div>
                <div :class="['ct-row', 'ct-sub-row', { 'ct-row-alt': (mi as number) % 2 === 0 }]">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-mount-from">
                    from {{ m.name }} ({{ m.readOnly ? 'ro' : 'rw' }})
                    <span v-if="m.subPath" class="ct-mount-sub">subPath={{ m.subPath }}</span>
                  </span>
                </div>
              </template>
            </template>

            <div class="ct-row">
              <span class="ct-label">Command</span>
              <span class="ct-val">{{ formatCommand(c.command) }}</span>
            </div>

            <div class="ct-row ct-row-alt" v-if="c.args && c.args.length">
              <span class="ct-label">Args</span>
              <span class="ct-val">{{ formatCommand(c.args) }}</span>
            </div>

            <div class="ct-row" v-if="c.workingDir">
              <span class="ct-label">WorkingDir</span>
              <span class="ct-val">{{ c.workingDir }}</span>
            </div>

            <div :class="['ct-row', { 'ct-row-alt': !c.workingDir }]">
              <span class="ct-label">Requests</span>
              <span class="ct-val">
                CPU: {{ c.resources?.requestsCpu || '—' }}, Memory: {{ c.resources?.requestsMemory || '—' }}
              </span>
            </div>

            <div class="ct-row ct-row-alt">
              <span class="ct-label">Limits</span>
              <span class="ct-val">
                CPU: {{ c.resources?.limitsCpu || '—' }}, Memory: {{ c.resources?.limitsMemory || '—' }}
              </span>
            </div>

            <template v-if="c.probes">
              <div class="ct-row ct-row-click" @click.stop="toggleContainerSection(c.name, 'probes')">
                <span class="ct-label">
                  Probes
                  <span class="ct-sec-toggle">{{ isContainerSectionCollapsed(c.name, 'probes') ? '›' : '×' }}</span>
                </span>
                <span class="ct-val" v-if="isContainerSectionCollapsed(c.name, 'probes')">Configured</span>
              </div>
              <template v-if="!isContainerSectionCollapsed(c.name, 'probes')">
                <div class="ct-row ct-sub-row">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">Liveness</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ c.probes.liveness || '—' }}</span>
                  </span>
                </div>
                <div class="ct-row ct-sub-row ct-row-alt">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">Readiness</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ c.probes.readiness || '—' }}</span>
                  </span>
                </div>
                <div class="ct-row ct-sub-row">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">Startup</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ c.probes.startup || '—' }}</span>
                  </span>
                </div>
              </template>
            </template>

            <template v-if="c.securityContext">
              <div class="ct-row ct-row-alt ct-row-click" @click.stop="toggleContainerSection(c.name, 'security')">
                <span class="ct-label">
                  Security Context
                  <span class="ct-sec-toggle">{{ isContainerSectionCollapsed(c.name, 'security') ? '›' : '×' }}</span>
                </span>
                <span class="ct-val" v-if="isContainerSectionCollapsed(c.name, 'security')">Configured</span>
              </div>
              <template v-if="!isContainerSectionCollapsed(c.name, 'security')">
                <div class="ct-row ct-sub-row">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">RunAsUser</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ fmtMaybe(c.securityContext.runAsUser) }}</span>
                  </span>
                </div>
                <div class="ct-row ct-sub-row ct-row-alt">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">RunAsNonRoot</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ fmtMaybe(c.securityContext.runAsNonRoot) }}</span>
                  </span>
                </div>
                <div class="ct-row ct-sub-row">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">ReadOnlyFilesystem</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ fmtMaybe(c.securityContext.readOnlyFilesystem) }}</span>
                  </span>
                </div>
                <div class="ct-row ct-sub-row ct-row-alt">
                  <span class="ct-label"></span>
                  <span class="ct-val ct-env-cell">
                    <span class="ct-env-name">AllowPrivilegeEscalation</span>
                    <span class="ct-env-sep">:</span>
                    <span class="ct-env-value">{{ fmtMaybe(c.securityContext.allowPrivilegeEscalation) }}</span>
                  </span>
                </div>
              </template>
            </template>

            <div class="ct-row">
              <span class="ct-label">TerminationMessagePath</span>
              <span class="ct-val">{{ c.terminationMessagePath || '—' }}</span>
            </div>
          </div>
        </div>
      </div>
    </template>

    <!-- Events tab -->
    <template #events>
      <div class="empty-section">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'
import type {
  PortForwardSession,
  PodAntiAffinity,
  ConditionInfo,
  VolumeInfo,
} from '../../../types'
import K8sResourceDetailBase from '../K8sResourceDetailBase.vue'
import PodMetricsViewer from '../../metrics/PodMetricsViewer.vue'
import ContainerMetricsViewer from '../../metrics/ContainerMetricsViewer.vue'
import PodPortsSection from '../PodPortsSection.vue'
import { useKubPodActions } from '../../../composables/KubPodActions'
import attachIcon from '../../../assets/icons/attach.svg'
import terminalIcon from '../../../assets/icons/terminal.svg'
import logsIcon from '../../../assets/icons/logs.svg'
import evictIcon from '../../../assets/icons/evict.svg'

interface DetailTab {
  id: string
  label: string
}

const props = defineProps<{
  pod: any
  statusClass: (s: string) => string
  clusterName?: string
  portForwards?: any[]
  onStartPortForward?: (ns: string, podName: string, localPort: number, podPort: number) => Promise<void>
  onStopPortForward?: (id: string) => Promise<void>
}>()

const emit = defineEmits<{
  (e: 'close'): void
  (e: 'delete'): void
  (e: 'attach', data: { container: string; pod: any }): void
  (e: 'terminal', data: { container: string; pod: any }): void
  (e: 'evict'): void
  (e: 'logs', data: { container: string; pod: any }): void
  (e: 'edit'): void
  (e: 'open-resource', data: { kind: string; name: string; namespace?: string }): void
}>()

// ═══════ Используем общий композабл для действий ═══════

const selectedPod = computed(() => props.pod)

const podActions = {
  openAttachWithContainer: (container: string, pod: any) => {
    showAttachMenu.value = false
    emit('attach', { container, pod })
  },
  openShellWithContainer: (container: string, pod: any) => {
    showTerminalMenu.value = false
    emit('terminal', { container, pod })
  },
  openLogs: (pod: any, container?: string) => {
    showLogsMenu.value = false
    emit('logs', { container: container || '', pod })
  },
  evictPod: (pod: any) => {
    emit('evict')
  }
}

const k8sActions = {
  showYAML: (kind: string, namespace: string, name: string) => {
    emit('edit')
  },
  refreshData: () => {},
  deleteResource: (namespace: string, name: string) => {
    emit('delete')
  }
}

const { handleContextAction } = useKubPodActions(
  selectedPod,
  podActions,
  k8sActions
)

// ═══════ Action menu state ═══════

const showAttachMenu = ref(false)
const showTerminalMenu = ref(false)
const showLogsMenu = ref(false)

function handleAttach(container: string) {
  handleContextAction(`attachPod:${container}`, props.pod)
}

function handleTerminal(container: string) {
  handleContextAction(`shell:${container}`, props.pod)
}

function handleLogs(container: string) {
  handleContextAction(`logs:${container}`, props.pod)
}

function openResource(kind: string, name: string, namespace?: string) {
  const data = {
    kind,
    name,
    namespace: namespace || props.pod.namespace
  }
  emit('open-resource', data)
}

function getControlledByType(): string {
  if (!props.pod.controlledBy) return ''
  const val = props.pod.controlledBy
  if (val.includes('/')) {
    const parts = val.split('/')
    return parts[0]
  }
  return ''
}

function getControlledByName(): string {
  if (!props.pod.controlledBy) return ''
  const val = props.pod.controlledBy
  if (val.includes('/')) {
    const parts = val.split('/')
    return parts[parts.length - 1]
  }
  return val
}

function openControlledByResource() {
  if (!props.pod.controlledBy) return
  const val = props.pod.controlledBy
  let kind = 'ReplicaSet'
  let name = val
  if (val.includes('/')) {
    const parts = val.split('/')
    if (parts.length === 2) {
      kind = parts[0]
      name = parts[1]
    }
  }
  openResource(kind, name, props.pod.namespace)
}

// ═══════ Custom tabs ═══════

const customTabs = computed<DetailTab[]>(() => {
  const tabs: DetailTab[] = []
  if (props.pod.containers && props.pod.containers.length > 0) {
    tabs.push({ id: 'containers', label: 'Containers' })
  }
  return tabs
})

// ═══════ Properties expand state ═══════

const metricsExpanded = ref(false)
const labelsExpanded = ref(false)
const annotationsExpanded = ref(false)
const tolerationsExpanded = ref(false)
const affinityExpanded = ref(false)
const volumesExpanded = ref(false)

// ═══════ Affinity ═══════

const affinityRuleCount = computed(() => {
  const a = props.pod.podAntiAffinity as PodAntiAffinity | undefined
  if (!a) return 0
  let count = 0
  if (a.preferredDuringSchedulingIgnoredDuringExecution?.length) count++
  if (a.requiredDuringSchedulingIgnoredDuringExecution?.length) count++
  return count
})

// ═══════ Containers UI state ═══════

const collapsedContainers = ref<Set<string>>(new Set())
const collapsedContainerMetrics = ref<Set<string>>(new Set())

type ContainerSection = 'ports' | 'env' | 'mounts' | 'probes' | 'security'
const collapsedContainerSections = ref<Map<string, Set<ContainerSection>>>(new Map())

function toggleContainer(name: string) {
  const s = new Set(collapsedContainers.value)
  if (s.has(name)) s.delete(name)
  else s.add(name)
  collapsedContainers.value = s
}

function toggleContainerMetrics(containerName: string) {
  const s = new Set(collapsedContainerMetrics.value)
  if (s.has(containerName)) s.delete(containerName)
  else s.add(containerName)
  collapsedContainerMetrics.value = s
}

function getSectionSet(containerName: string): Set<ContainerSection> {
  const current = collapsedContainerSections.value.get(containerName)
  if (current) return current
  const init = new Set<ContainerSection>(['ports', 'env', 'mounts', 'probes', 'security'])
  collapsedContainerSections.value.set(containerName, init)
  return init
}

function isContainerSectionCollapsed(containerName: string, section: ContainerSection): boolean {
  return getSectionSet(containerName).has(section)
}

function toggleContainerSection(containerName: string, section: ContainerSection) {
  const map = new Map(collapsedContainerSections.value)
  const set = new Set(getSectionSet(containerName))
  if (set.has(section)) set.delete(section)
  else set.add(section)
  map.set(containerName, set)
  collapsedContainerSections.value = map
}

function containerStatusColor(c: any): string {
  if (c.state === 'running' && c.ready) return 'ct-status-ok'
  if (c.state === 'running') return 'ct-status-warn'
  if (['CrashLoopBackOff', 'Error', 'OOMKilled'].includes(c.state)) return 'ct-status-err'
  return 'ct-status-neutral'
}

// ═══════ Helpers ═══════

function formatCommand(cmd: string[] | undefined): string {
  if (!cmd || !cmd.length) return '—'
  return cmd.join(' ')
}

function fmtMaybe(v: unknown): string {
  if (v === null || v === undefined) return '—'
  return String(v)
}

function formatMap(o: Record<string, string> | undefined): string {
  if (!o || !Object.keys(o).length) return 'None'
  return Object.entries(o).map(([k, v]) => k + '=' + v).join('\n')
}

function formatConditions(c: ConditionInfo[]): string {
  if (!c || !c.length) return 'None'
  return c.map((x) => x.type).join(' ')
}

function esc(s: string): string {
  return s.replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')
}

// ═══════ YAML tree (affinity editor) ═══════

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
    while (stack.length && stack[stack.length - 1].indent >= node.indent) stack.pop()
    if (!stack.length) roots.push(node)
    else stack[stack.length - 1].children.push(node)
    if (node.raw.trimEnd().endsWith(':')) stack.push(node)
  }
  return roots
}

const collapsedSet = ref<Set<number>>(new Set())

function toggleFold(flatIdx: number) {
  const s = new Set(collapsedSet.value)
  if (s.has(flatIdx)) s.delete(flatIdx)
  else s.add(flatIdx)
  collapsedSet.value = s
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
        if (lines[j].indent > level) {
          guides.push(level)
          break
        }
      }
    }
    lines[i].activeGuides = guides
  }
}

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
      for (const [k, v] of Object.entries(ml)) out.push(sp(5) + k + ': ' + v)
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
      for (const [k, v] of Object.entries(ml)) out.push(sp(3) + k + ': ' + v)
      out.push(sp(2) + 'topologyKey: ' + t.topologyKey)
    })
  }
  return out
}

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
      for (const key of Object.keys(item.details)) colSet.add(key)
    }
    groups.push({ type: formatTypeName(type), items, columns: Array.from(colSet) })
  }
  return groups
})

function formatTypeName(type: string): string {
  return type.replace(/([a-z])([A-Z])/g, '$1 $2')
}
</script>
<style scoped>
/* ═══════ Two-level action menu ═══════ */

.action-group {
  position: relative;
  display: inline-flex;
  align-items: center;
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
  white-space: nowrap;
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

.action-dropdown {
  position: absolute;
  top: 100%;
  left: 0;
  margin-top: 4px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 150px;
  z-index: 10;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
}

.action-dropdown.empty {
  padding: 8px 12px;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.dropdown-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
}

.dropdown-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.dropdown-item:first-child {
  border-radius: 3px 3px 0 0;
}

.dropdown-item:last-child {
  border-radius: 0 0 3px 3px;
}

/* ═══════ Button Icons ═══════ */

.btn-icon {
  width: 24px;
  height: 24px;
  display: inline-block;
  vertical-align: middle;
}

.btn-action .btn-icon {
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

/* ═══════ Pod Properties ═══════ */

.pod-properties {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.meta-row {
  display: grid;
  grid-template-columns: auto 1fr;
  align-items: baseline;
  margin-bottom: 8px;
  gap: 12px;
  text-align: left;
}

.meta-label {
  font-weight: bold;
  color: #d4d4d4;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.meta-content {
  margin: 0;
  padding: 0;
  text-align: left;
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

.status-badge {
  display: inline-block;
  padding: 2px 8px;
  background: rgba(59, 130, 246, 0.2);
  color: #60a5fa;
  border-radius: 3px;
  font-size: 0.8rem;
  font-weight: 600;
}

.tolerations-table {
  width: 100%;
  margin-top: 8px;
}

.tolerations-table th,
.tolerations-table td {
  text-align: left;
  padding: 4px 8px;
  font-size: 0.8rem;
}

.tolerations-table th {
  font-weight: bold;
  border-bottom: 1px solid #333;
}

.tolerations-table td {
  border-bottom: 1px solid #2a2b40;
}

/* ═══════ Affinity Editor ═══════ */

.affinity-editor-wrapper {
  margin-top: 4px;
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
  /* Using universal scrollbar styles from K8sResourceDetailBase */
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

/* ═══════ Pod Volumes ═══════ */

.volumes-wrapper {
  margin-bottom: 12px;
}

.volumes-empty {
  color: #6b6d8a;
  padding: 4px 0;
}

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

/* ═══════ Containers Section ═══════ */

.containers-section {
  display: flex;
  flex-direction: column;
  gap: 10px;
}

.ct-block {
  border: 1px solid #2a2b40;
  border-radius: 6px;
  overflow: hidden;
  background: #1a1b2e;
}

/* Container Header */

.ct-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  background: #1e1f35;
  border-bottom: 1px solid #2a2b40;
  user-select: none;
  gap: 12px;
}

.ct-header-left {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}

.ct-chevron {
  display: inline-block;
  font-size: 14px;
  color: #6b6d8a;
  transition: transform 0.15s ease;
  transform: rotate(0deg);
  flex-shrink: 0;
  width: 16px;
  text-align: center;
}

.ct-chevron.expanded {
  transform: rotate(90deg);
}

.ct-name {
  font-weight: 600;
  color: #e0e2f0;
  font-size: 0.95rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.ct-badge-init {
  font-size: 0.7rem;
  padding: 1px 6px;
  border-radius: 3px;
  background: #3b3d5c;
  color: #a0a2c0;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.5px;
  flex-shrink: 0;
}

.ct-header-image {
  color: #6b6d8a;
  font-size: 0.82rem;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  max-width: 400px;
  text-align: right;
}

/* Container Body */

.ct-body {
  font-size: 0.875rem;
}

/* Container Rows */

.ct-row {
  display: grid;
  grid-template-columns: 160px 1fr;
  align-items: baseline;
  padding: 6px 14px;
  min-height: 32px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.03);
}

.ct-row-alt {
  background: rgba(255, 255, 255, 0.02);
}

.ct-row-click {
  cursor: pointer;
}

.ct-row-click:hover {
  background: rgba(255, 255, 255, 0.04);
}

.ct-sub-row {
  padding-left: 14px;
}

.ct-label {
  color: #9ca0b8;
  font-weight: 600;
  font-size: 0.82rem;
  white-space: nowrap;
  user-select: none;
  display: flex;
  align-items: center;
  gap: 4px;
}

.ct-val {
  color: #d4d4d4;
  min-width: 0;
  overflow-wrap: anywhere;
  word-break: break-word;
  text-align: left;
}

.ct-sec-toggle {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 16px;
  height: 16px;
  font-size: 13px;
  color: #6b6d8a;
  border-radius: 3px;
  transition: color 0.15s;
  flex-shrink: 0;
}

.ct-sec-toggle:hover {
  color: #a0a2c0;
}

/* Status colors */

.ct-status-ok {
  color: #34d399;
}

.ct-status-warn {
  color: #fbbf24;
}

.ct-status-err {
  color: #f87171;
}

.ct-status-neutral {
  color: #9ca0b8;
}

/* Environment */

.ct-env-cell {
  display: flex;
  align-items: baseline;
  gap: 4px;
  min-width: 0;
}

.ct-env-name {
  color: #c0c2d8;
  font-weight: 500;
  white-space: nowrap;
  flex-shrink: 0;
}

.ct-env-sep {
  color: #4a4b6a;
  flex-shrink: 0;
}

.ct-env-value {
  color: #9ca0b8;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  min-width: 0;
}

/* Mounts */

.ct-mount-path {
  color: #d4d4d4;
  font-family: 'JetBrains Mono', 'Fira Code', monospace;
  font-size: 0.82rem;
  background: rgba(255, 255, 255, 0.04);
  padding: 2px 8px;
  border-radius: 3px;
  display: inline-block;
}

.ct-mount-from {
  color: #9ca0b8;
  font-size: 0.82rem;
  padding-left: 8px;
}

.ct-mount-sub {
  color: #6b6d8a;
  margin-left: 6px;
}

/* Metrics Section */

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

.ct-metrics-wrapper {
  width: 100%;
  margin-bottom: 12px;
  border: 1px solid #2a2b40;
  border-radius: 6px;
  overflow: hidden;
  background: #1a1b2e;
}

.ct-metrics-header {
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

.ct-metrics-header:hover {
  background: linear-gradient(135deg, rgba(59, 130, 246, 0.15) 0%, rgba(59, 130, 246, 0.1) 100%);
}

.ct-metrics-title {
  flex: 1;
  font-weight: 600;
  color: #d4d4d4;
  font-size: 0.95rem;
  letter-spacing: 0.3px;
}

.ct-metrics-toggle {
  display: inline-block;
  width: 20px;
  height: 20px;
  flex-shrink: 0;
  position: relative;
  transition: transform 0.3s cubic-bezier(0.4, 0, 0.2, 1);
}

.ct-metrics-toggle::before {
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

.ct-metrics-header:hover .ct-metrics-toggle::before {
  border-color: #a0a2c0;
}

.ct-metrics-toggle.collapsed {
  transform: rotate(-90deg);
}

.ct-metrics-content {
  padding: 16px;
  max-height: 600px;
  overflow-y: auto;
}

.ct-metrics-section {
  width: 100%;
  margin-bottom: 12px;
  padding: 0 14px;
}

.empty-section {
  padding: 16px;
  text-align: center;
  color: #6b6d8a;
  font-size: 0.875rem;
}

.cursor-pointer {
  cursor: pointer;
}

/* ═══════ Status Text ═══════ */

.status-text {
  font-weight: 500;
}

.status-text.status-running {
  color: #34d399;
}

.status-text.status-pending {
  color: #fbbf24;
}

.status-text.status-error {
  color: #f87171;
}

.status-text.status-completed {
  color: #34d399;
}

.status-text.status-unknown {
  color: #9ca0b8;
}

/* ═══════ Controlled By ═══════ */

.controlled-by-wrapper {
  display: inline-flex;
  align-items: baseline;
  gap: 4px;
}

.controlled-by-type {
  color: #d4d4d4;
  font-weight: 500;
  white-space: nowrap;
}

/* ═══════ Meta Links ═══════ */

.meta-link {
  color: #60a5fa;
  cursor: pointer;
  text-decoration: underline;
  text-decoration-color: #60a5fa;
  text-decoration-thickness: 1px;
  text-underline-offset: 2px;
  transition: all 0.15s ease;
  display: inline;
}

.meta-link:hover {
  color: #93c5fd;
  text-decoration-color: #93c5fd;
}

.meta-link:active {
  color: #3b82f6;
  text-decoration-color: #3b82f6;
}
</style>
