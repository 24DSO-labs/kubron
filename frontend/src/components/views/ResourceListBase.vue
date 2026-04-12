<template>
  <div>
    <div v-if="showContextMenu" class="menu-overlay" @click="hideContextMenu"></div>
    <ContentHeader icon="" :title="resourceTitle" :count="items.length" :search="search" showSearch @refresh="onRefresh" @search="onSearch" />
    <table v-if="items.length > 0" class="table">
      <thead>
        <tr>
          <th v-for="col in mergedColumns" :key="col.key">{{ col.label }}</th>
          <th></th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="item in items" :key="getItemKey(item)" @click="handleRowClick(item)" class="clickable-row">
  <td v-for="col in mergedColumns" :key="col.key" :class="col.className">
    <component v-if="col.component" :is="col.component" :item="item" :col="col" />
    <span v-else :style="getColumnStyle(col, item)">
      {{ getColumnValue(col, item) }}
    </span>
  </td>
          <td @click.stop>
            <button class="btn-icon" @click.stop="openContextMenu(item, $event)">⋮</button>
          </td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">{{ emptyMessage }}</div>
    <div v-if="showContextMenu" :style="menuStyle" class="cluster-context-menu" @click.stop>
      <template v-for="(item, index) in mergedMenuItems" :key="index">
        <div
          v-if="item.action === 'shell'"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="() => { cancelShellMenuTimeout(); showShellMenu = true }"
          @mouseleave="scheduleHideShellMenu"
        >
          <span class="ctx-menu-arrow" v-if="hasShellSubmenu">></span>{{ item.label }}
          <div v-if="showShellMenu && hasShellSubmenu" class="ctx-menu-sub">
            <div v-for="container in shellContainers" :key="container.name" class="ctx-menu-sub-item" @click="handleContextAction({ action: `shell:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        <div
          v-else-if="item.action === 'attach'"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="() => { cancelAttachMenuTimeout(); showAttachMenu = true }"
          @mouseleave="scheduleHideAttachMenu"
        >
          <span class="ctx-menu-arrow" v-if="hasAttachSubmenu">></span>{{ item.label }}
          <div v-if="showAttachMenu && hasAttachSubmenu" class="ctx-menu-sub">
            <div v-for="container in attachContainers" :key="container.name" class="ctx-menu-sub-item" @click="handleContextAction({ action: `attach:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        <div
          v-else-if="item.action === 'logs'"
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @mouseenter="() => { cancelLogsMenuTimeout(); showLogsMenu = true }"
          @mouseleave="scheduleHideLogsMenu"
        >
          <span class="ctx-menu-arrow" v-if="hasLogsSubmenu">></span>{{ item.label }}
          <div v-if="showLogsMenu && hasLogsSubmenu" class="ctx-menu-sub">
            <div v-for="container in logsContainers" :key="container.name" class="ctx-menu-sub-item" @click="handleContextAction({ action: `logs:${container.name}` })">
              {{ container.name }}
            </div>
          </div>
        </div>
        <div
          v-else
          :class="['ctx-menu-item', { 'ctx-menu-danger': item.danger }]"
          @click="handleContextAction({ action: item.action })"
        >
          <span class="ctx-menu-arrow"></span>{{ item.label }}
        </div>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import ContentHeader from '../ContentHeader.vue'
import { ref, onUnmounted } from 'vue'
import useResourceListBase from '../../composables/useResourceListBase'
import { useResourceDetailOpener } from '../../composables/useDetailPanel'
import { useK8s } from '../../composables/useK8s'
import type { Column, ContextMenuItem } from '../../types'

const props = defineProps<{
  items: any[]
  search: string
  resourceTitle: string
  emptyMessage: string
  resourceType: string // Новый prop для типа ресурса
  customColumns?: Column[] // Опциональные кастомные столбцы
  customMenuItems?: ContextMenuItem[] // Опциональные кастомные меню
  getItemKey?: (item: any) => string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
  (e: 'select', item: any): void
  (e: 'contextAction', action: string, item: any, subAction?: string): void
  (e: 'contextOpen', item: any): void
  (e: 'shell:open', namespace: string, pod: string, container: string): void
  (e: 'attach:open', namespace: string, pod: string, container: string): void
  (e: 'logs:open', namespace: string, pod: string, container: string): void
}>()

// Используем composable для defaults
const { mergedColumns, mergedMenuItems } = useResourceListBase(props.resourceType)

// Используем composable для автоматического открытия деталей
const { openDetails } = useResourceDetailOpener()

// Слияние с кастомными из props, если предоставлены
const finalColumns = props.customColumns ? [...mergedColumns, ...props.customColumns] : mergedColumns
const finalMenuItems = props.customMenuItems ? [...mergedMenuItems, ...props.customMenuItems] : mergedMenuItems

const showContextMenu = ref(false)
const menuStyle = ref<Record<string, string>>({})
const selectedItem = ref<any>(null)
const localContextMenuItems = ref<ContextMenuItem[]>([])
const showAttachMenu = ref(false)
const showShellMenu = ref(false)
const showLogsMenu = ref(false)

const attachContainers = ref<any[]>([])
const shellContainers = ref<any[]>([])
const logsContainers = ref<any[]>([])
const hasAttachSubmenu = ref(false)
const hasShellSubmenu = ref(false)
const hasLogsSubmenu = ref(false)

let attachMenuTimeout: ReturnType<typeof setTimeout> | null = null
let shellMenuTimeout: ReturnType<typeof setTimeout> | null = null
let logsMenuTimeout: ReturnType<typeof setTimeout> | null = null

function scheduleHideAttachMenu() {
  if (attachMenuTimeout) clearTimeout(attachMenuTimeout)
  attachMenuTimeout = setTimeout(() => {
    showAttachMenu.value = false
  }, 150)
}

function scheduleHideShellMenu() {
  if (shellMenuTimeout) clearTimeout(shellMenuTimeout)
  shellMenuTimeout = setTimeout(() => {
    showShellMenu.value = false
  }, 150)
}

function scheduleHideLogsMenu() {
  if (logsMenuTimeout) clearTimeout(logsMenuTimeout)
  logsMenuTimeout = setTimeout(() => {
    showLogsMenu.value = false
  }, 150)
}

function cancelAttachMenuTimeout() {
  if (attachMenuTimeout) clearTimeout(attachMenuTimeout)
  attachMenuTimeout = null
}

function cancelShellMenuTimeout() {
  if (shellMenuTimeout) clearTimeout(shellMenuTimeout)
  shellMenuTimeout = null
}

function cancelLogsMenuTimeout() {
  if (logsMenuTimeout) clearTimeout(logsMenuTimeout)
  logsMenuTimeout = null
}

onUnmounted(() => {
  cancelAttachMenuTimeout()
  cancelShellMenuTimeout()
  cancelLogsMenuTimeout()
})

function getCellValue(item: any, key: string): string {
  const keys = key.split('.')
  let value: any = item
  for (const k of keys) {
    value = value?.[k]
  }
  return value ?? '—'
}

function getStatusColor(status: string): string {
  const statusLower = status?.toLowerCase() || ''
  
  if (statusLower === 'running') return '#4ade80'
  if (statusLower === 'succeeded') return '#4ade80'
  if (statusLower === 'pending') return '#facc15'
  if (statusLower === 'failed') return '#ef4444'
  if (statusLower === 'crashed' || statusLower === 'crashloopbackoff') return '#dc2626'
  if (statusLower === 'imagepullbackoff') return '#f97316'
  if (statusLower === 'evicted') return '#ef4444'
  if (statusLower === 'terminating') return '#f97316'
  if (statusLower === 'unknown') return '#868d8a'
  if (statusLower === 'progressdeadlineexceeded') return '#ec4899'
  if (statusLower === 'replicafailure') return '#ef4444'
  
  return '#868d8a'
}

function getColumnValue(col: Column, item: any): string {
  if (col.formatter) return col.formatter(item)
  return getCellValue(item, col.key)
}

function getColumnStyle(col: Column, item: any): { color?: string } {
  if (col.colorFn) return { color: col.colorFn(item) }
  if (col.key === 'status') return { color: getStatusColor(getCellValue(item, col.key)) }
  return {}
}

function getItemKey(item: any): string {
  if (props.getItemKey) return props.getItemKey(item)
  const ns = (item as any).namespace
  return ns ? `$${ns}$${item.name}` : item.name
}

function openContextMenu(item: any, event: MouseEvent) {
  selectedItem.value = item
  emit('contextOpen', item)
  
  localContextMenuItems.value = JSON.parse(JSON.stringify(finalMenuItems))
  
  localContextMenuItems.value.forEach(menuItem => {
    menuItem.show = false
  })
  
  const containers = (item as any).containers || []
  const regularContainers = containers.filter((c: any) => !c.isInit)
  
  attachContainers.value = regularContainers
  shellContainers.value = regularContainers
  logsContainers.value = regularContainers
  hasAttachSubmenu.value = regularContainers.length > 0
  hasShellSubmenu.value = regularContainers.length > 0
  hasLogsSubmenu.value = regularContainers.length > 0
  
  // Если нет контейнеров (не под), используем имя ресурса как идентификатор для логов
  if (regularContainers.length === 0) {
    const resourceName = item.name || ''
    logsContainers.value = [{ name: resourceName }]
    hasLogsSubmenu.value = false
  }
  
  showAttachMenu.value = false
  showShellMenu.value = false
  showLogsMenu.value = false

  const button = event.target as HTMLElement
  const rect = button.getBoundingClientRect()
  const menuWidth = 200
  const menuHeight = Math.min(300, finalMenuItems.length * 30 + 50)

  let left
  const availableLeft = rect.left
  if (availableLeft >= menuWidth) {
    left = rect.left - menuWidth + window.scrollX
  } else {
    left = rect.right + window.scrollX
    if (left + menuWidth > window.innerWidth + window.scrollX) {
      left = window.innerWidth - menuWidth + window.scrollX
    }
  }

  let top
  const availableBottom = window.innerHeight - rect.bottom
  if (availableBottom >= menuHeight) {
    top = rect.bottom + window.scrollY
  } else {
    top = rect.top - menuHeight + window.scrollY
    if (top < window.scrollY) {
      top = window.scrollY
    }
  }

  menuStyle.value = {
    position: 'absolute',
    top: `${top}px`,
    left: `${left}px`
  }
  showContextMenu.value = true
}

function hideContextMenu() {
  showContextMenu.value = false
}

async function handleContextAction(menuItem: any, subAction?: string) {
  console.log('[ResourceListBase] handleContextAction вызван', { menuItem, subAction, selectedItem: selectedItem.value })
  
  if (selectedItem.value && menuItem.action) {
    const action = menuItem.action
    const item = selectedItem.value
    const namespace = item.namespace || ''
    const podName = item.name
    const container = subAction || item.container || ''
    
    console.log('[ResourceListBase] Обработка действия:', { action, namespace, podName, container })
    
    if (action === 'details') {
      console.log('[ResourceListBase] Открытие деталей')
      openDetails(props.resourceType, selectedItem.value)
    } else if (action === 'delete') {
      console.log('[ResourceListBase] Удаление элемента')
      await handleDelete(selectedItem.value)
    } else if (action.startsWith('shell:')) {
      const shellContainer = action.split(':')[1] || container
      console.log('[ResourceListBase] Эмит shell:open', { namespace, podName, shellContainer })
      emit('shell:open', namespace, podName, shellContainer)
    } else if (action.startsWith('attach:')) {
      const attachContainer = action.split(':')[1] || container
      console.log('[ResourceListBase] Эмит attach:open', { namespace, podName, attachContainer })
      emit('attach:open', namespace, podName, attachContainer)
    } else if (action.startsWith('logs:')) {
      const logsContainer = action.split(':')[1] || container
      console.log('[ResourceListBase] Эмит logs:open', { namespace, podName, logsContainer })
      emit('logs:open', namespace, podName, logsContainer)
    } else {
      console.log('[ResourceListBase] Эмит contextAction', { action, item })
      emit('contextAction', action, selectedItem.value)
    }
  }
  hideContextMenu()
}

async function handleDelete(item: any) {
  if (!confirm(`Удалить ${props.resourceType} ${item.name}?`)) return
  
  try {
    const k8s = useK8s()
    const namespace = item.namespace || ''
    const name = item.name
    
    // Вызываем соответствующую функцию delete на основе типа ресурса
    const resourceTypeLower = props.resourceType.toLowerCase()
    
    switch (resourceTypeLower) {
      case 'pod':
        await k8s.deletePodByName(namespace, name)
        break
      case 'deployment':
        await k8s.deleteDeploymentByName(namespace, name)
        break
      case 'statefulset':
        console.log(`Delete StatefulSet: ${namespace}/${name}`)
        emit('contextAction', 'deleteStatefulSet', item)
        break
      case 'daemonset':
        console.log(`Delete DaemonSet: ${namespace}/${name}`)
        emit('contextAction', 'deleteDaemonSet', item)
        break
      case 'job':
        console.log(`Delete Job: ${namespace}/${name}`)
        emit('contextAction', 'deleteJob', item)
        break
      case 'cronjob':
        console.log(`Delete CronJob: ${namespace}/${name}`)
        emit('contextAction', 'deleteCronJob', item)
        break
      case 'service':
        console.log(`Delete Service: ${namespace}/${name}`)
        emit('contextAction', 'deleteService', item)
        break
      case 'configmap':
        console.log(`Delete ConfigMap: ${namespace}/${name}`)
        emit('contextAction', 'deleteConfigMap', item)
        break
      case 'secret':
        console.log(`Delete Secret: ${namespace}/${name}`)
        emit('contextAction', 'deleteSecret', item)
        break
      case 'serviceaccount':
        await k8s.deleteServiceAccount(namespace, name)
        break
      case 'role':
        await k8s.deleteRole(namespace, name)
        break
      case 'rolebinding':
        await k8s.deleteRoleBinding(namespace, name)
        break
      case 'clusterrole':
        await k8s.deleteClusterRole(name)
        break
      case 'clusterrolebinding':
        await k8s.deleteClusterRoleBinding(name)
        break
      default:
        console.error(`Delete not implemented for resource type: ${props.resourceType}`)
        throw new Error(`Delete не поддерживается для типа ресурса: ${props.resourceType}`)
    }
    
    // После успешного удаления эмитим событие refresh
    emit('refresh')
  } catch (e: any) {
    const message = e.message || String(e)
    console.error(`Ошибка удаления ${props.resourceType}:`, message)
    alert(`Ошибка удаления: ${message}`)
  }
}

function handleRowClick(item: any) {
  // Сначала эмитим событие select для обратной совместимости
  emit('select', item)
  
  // Затем автоматически открываем детальную панель
  openDetails(props.resourceType, item)
}

function onRefresh() { emit('refresh') }
function onSearch(val: string) { emit('search', val) }
</script>

<style scoped>
/* Скопировать стиль из оригинального K8sResourceListBase.vue */
.menu-overlay {
  position: fixed;
  top: 0;
  left: 0;
  width: 100%;
  height: 100%;
  z-index: 9998;
  cursor: default;
}

.cluster-context-menu {
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 9999;
  min-width: 160px;
  overflow: visible;
  padding: 0;
  position: relative;
}

.ctx-menu-item {
  padding: 6px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  padding-left: 34px;
  text-align: left;
  position: relative;
}

.ctx-menu-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-arrow {
  position: absolute;
  left: 19px;
  color: #868d8a;
  font-size: 0.9rem;
}

.ctx-menu-sub {
  position: absolute;
  top: 0;
  right: 100%;
  margin-right: 4px;
  background: #1e1f35;
  border: 1px solid #3b3d5c;
  border-radius: 4px;
  min-width: 160px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  z-index: 10000;
  white-space: nowrap;
}

.ctx-menu-sub-item {
  padding: 8px 12px;
  color: #d4d4d4;
  cursor: pointer;
  font-size: 0.875rem;
  transition: background 0.15s;
  white-space: nowrap;
}

.ctx-menu-sub-item:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.ctx-menu-danger {
  color: #ef4444;
}

.ctx-menu-danger:hover {
  background: rgba(239, 68, 68, 0.2);
}

.clickable-row {
  cursor: pointer;
}

.clickable-row:hover {
  background: rgba(255, 255, 255, 0.02);
}

.btn-icon {
  background: transparent;
  border: none;
  color: #d4d4d4;
  cursor: pointer;
  padding: 4px 8px;
  border-radius: 4px;
  transition: background 0.15s;
}

.btn-icon:hover {
  background: rgba(255, 255, 255, 0.1);
  color: white;
}

.empty {
  padding: 20px;
  text-align: center;
  color: #868d8a;
}

td {
  text-align: left;
}
</style>
