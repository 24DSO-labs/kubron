// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { ref, computed } from 'vue'
import { uid } from '../resources/uid'
import { callApp } from '../platform/api'
import { t } from '../i18n'

export type TabType = 'logs' | 'yaml' | 'terminal'

export interface BaseTab {
  id: string
  type: TabType
  resourceType: string
  resourceName: string
  namespace: string
  icon: string
  title: string
}

export interface LogsTab extends BaseTab {
  type: 'logs'
  selectedPodName?: string
  selectedContainer?: string
  rawLogs?: string
  loading: boolean
  pods: Array<{ name: string; containers?: Array<{ name: string }> }>
}

export interface YamlTab extends BaseTab {
  type: 'yaml'
  content: string
  originalContent: string
  saving?: boolean
  /** Перечитывается ли манифест из кластера */
  loading?: boolean
  /** edit — редактор, diff — просмотр изменений перед применением */
  view?: 'edit' | 'diff'
  /** create — манифест нового объекта (кнопка «+»): создаётся, а не заменяет существующий */
  mode?: 'edit' | 'create'
  /** Отображаемое имя типа для заголовка вкладки создания: 'Deployment' */
  kindLabel?: string
}

export interface TerminalTab extends BaseTab {
  type: 'terminal'
  mode: 'shell' | 'attach'
  pod: string
  podObj: any | null
  container: string
  containers: Array<{ name: string }>
}

export type Tab = LogsTab | YamlTab | TerminalTab

// Глобальное состояние
const tabs = ref<Tab[]>([])
const activeTabId = ref<string>('')

function openLogsTab(resourceType: string, resourceName: string, namespace: string, containers?: Array<{ name: string }>, selectedContainer?: string) {
  const existingTab = tabs.value.find(tab => 
    tab.type === 'logs' &&
    tab.resourceType === resourceType && 
    tab.resourceName === resourceName && 
    tab.namespace === namespace
  ) as LogsTab | undefined

  if (existingTab) {
    activeTabId.value = existingTab.id
    if (selectedContainer) existingTab.selectedContainer = selectedContainer
  } else {
    const id = `logs-${uid()}`
    const newTab: LogsTab = {
      id,
      type: 'logs',
      icon: '📋',
      title: `${resourceType}/${resourceName}`,
      resourceType,
      resourceName,
      namespace,
      selectedContainer,
      loading: false,
      pods: [],
    }

    if (resourceType.toLowerCase() === 'pod' && containers) {
      newTab.pods = [{ name: resourceName, containers }]
      newTab.selectedPodName = resourceName
      if (!selectedContainer && containers.length > 0) {
        newTab.selectedContainer = containers[0].name
      }
    }

    tabs.value.push(newTab)
    activeTabId.value = id
  }
}

async function openTerminalTab(
  resourceType: string,
  resourceName: string,
  namespace: string,
  pod: string,
  container: string,
  mode: 'shell' | 'attach' = 'shell'
) {
  const id = `terminal-${mode}-${uid()}`
  
  // Получаем информацию о поде для списка контейнеров
  let podContainers: Array<{ name: string }> = []
  let podObj: any = null
  
  try {
    const allPods = await callApp('GetPods', [namespace])
    podObj = allPods.find((p: any) => p.name === pod)
    if (podObj && podObj.containers) {
      podContainers = podObj.containers
    } else {
      // Fallback: используем текущий контейнер
      podContainers = [{ name: container }]
    }
  } catch (e) {
    console.error('Error getting pod info:', e)
    podContainers = [{ name: container }]
  }
  
  const newTab: TerminalTab = {
    id,
    type: 'terminal',
    icon: mode === 'shell' ? '🖥' : '📎',
    title: t(mode === 'shell' ? 'utils.tab.terminalTitle' : 'utils.tab.attachTitle', { target: `${pod}/${container}` }),
    resourceType,
    resourceName,
    namespace,
    mode,
    pod,
    podObj,
    container,
    containers: podContainers,
  }
  
  tabs.value.push(newTab)
  activeTabId.value = id
}

// Дополнительная перегрузка для совместимости
function openTerminalTabExtended(
  resourceType: string,
  resourceName: string,
  resourceNamespace: string,
  namespace: string,
  pod: string,
  container: string,
  mode: 'shell' | 'attach' = 'shell'
) {
  return openTerminalTab(resourceType, resourceName, namespace, pod, container, mode)
}

function openYamlTab(resourceType: string, resourceName: string, namespace: string, content: string) {
  const existingTab = tabs.value.find(tab => 
    tab.type === 'yaml' &&
    tab.resourceType === resourceType && 
    tab.resourceName === resourceName && 
    tab.namespace === namespace
  ) as YamlTab | undefined

  if (existingTab) {
    activeTabId.value = existingTab.id
    // Вкладку с несохранёнными правками не перезаписываем: актуальный YAML из кластера
    // пользователь получит кнопкой Reload
    const dirty = existingTab.content !== existingTab.originalContent
    if (!dirty && content !== existingTab.content) {
      existingTab.content = content
      existingTab.originalContent = content
    }
  } else {
    const id = `yaml-${uid()}`
    const newTab: YamlTab = {
      id,
      type: 'yaml',
      icon: '📄',
      title: `${resourceType}/${resourceName}`,
      resourceType,
      resourceName,
      namespace,
      content,
      originalContent: content,
      view: 'edit',
    }
    
    tabs.value.push(newTab)
    activeTabId.value = id
  }
}

/**
 * Вкладка с манифестом нового объекта. Каждый вызов открывает отдельную вкладку:
 * у черновика ещё нет имени, по которому её можно было бы найти.
 * resourceType — ключ типа (ResourceDefinition.id): с ним манифест уйдёт в CreateResourceYAML.
 */
function openCreateYamlTab(resourceType: string, kindLabel: string, content: string) {
  const id = `yaml-new-${uid()}`
  const newTab: YamlTab = {
    id,
    type: 'yaml',
    icon: '➕',
    title: t('utils.tab.newTitle', { kind: kindLabel }),
    resourceType,
    resourceName: '',
    // namespace задаёт сам манифест, поэтому в заголовке его не показываем
    namespace: '',
    content,
    originalContent: '',
    view: 'edit',
    mode: 'create',
    kindLabel,
  }
  tabs.value.push(newTab)
  activeTabId.value = id
}

function closeTab(tabId: string) {
  const index = tabs.value.findIndex(t => t.id === tabId)
  if (index !== -1) {
    tabs.value.splice(index, 1)
  }
  
  if (activeTabId.value === tabId && tabs.value.length > 0) {
    activeTabId.value = tabs.value[Math.max(0, index - 1)].id
  }
  
  if (tabs.value.length === 0) {
    activeTabId.value = ''
  }
}

function closeAllTabs() {
  tabs.value = []
  activeTabId.value = ''
}

function isOpen() {
  return tabs.value.length > 0
}

// Экспортируем глобальный объект
export const globalTabbedPanel = {
  tabs,
  activeTabId,
  openLogsTab,
  openYamlTab,
  openCreateYamlTab,
  openTerminalTab,
  closeTab,
  closeAllTabs,
  isOpen
}