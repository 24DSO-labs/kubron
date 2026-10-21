import { resources } from './registry'
import type { NavGroupId } from './types'

/** Заголовки групп и начальное состояние (свёрнута/развёрнута) */
export interface NavGroupMeta {
  title: string
  expanded: boolean
  /** Группа вложена в другую (рисуется внутри неё, в NAV_ORDER не указывается) */
  parent?: NavGroupId
  /** Группа показывается, только если выполнено условие (например, в кластере есть Argo CD) */
  requires?: 'argocd'
  /** Пункты-кастомные view (viewRegistry), которые стоят в группе перед пунктами ресурсов */
  views?: NavItem[]
}

export const NAV_GROUPS: Record<NavGroupId, NavGroupMeta> = {
  workloads: { title: 'Workloads', expanded: true },
  config: { title: 'Config', expanded: false },
  network: { title: 'Network', expanded: true },
  storage: { title: 'Storage', expanded: false },
  helm: { title: 'Helm', expanded: false },
  access: { title: 'Access Control', expanded: false },
  argocd: {
    title: 'Argo CD',
    expanded: true,
    requires: 'argocd',
    views: [{ view: 'argo-dashboard', label: 'Dashboard' }],
  },
  'argocd-settings': { title: 'Settings', expanded: false, parent: 'argocd' },
}

/**
 * Порядок бокового меню между Overview и Settings.
 * Элемент — id группы или view ресурса верхнего уровня (у него в `nav` нет group).
 */
const NAV_ORDER: string[] = [
  'nodes', 'events',
  'workloads', 'config', 'network', 'storage',
  'namespaces',
  'helm', 'access', 'argocd',
]

export interface NavItem { view: string; label: string; icon?: string }
export interface NavGroup {
  type: 'group'
  id: NavGroupId
  title: string
  icon?: string
  requires?: NavGroupMeta['requires']
  items: NavItem[]
  /** Вложенные группы (рисуются после пунктов) */
  children: NavGroup[]
}
export type NavEntry = { type: 'item'; view: string; label: string; icon?: string } | NavGroup

const isGroup = (id: string): id is NavGroupId => id in NAV_GROUPS

/** Меню строится из `nav` в определениях ресурсов: новый ресурс появляется в нём без правок Sidebar */
export function buildNav(): NavEntry[] {
  const entries: NavEntry[] = []

  for (const id of NAV_ORDER) {
    if (isGroup(id)) {
      const group = buildGroup(id)
      if (group) entries.push(group)
    } else {
      const r = resources.find(x => x.view === id && x.nav && !x.nav.group)
      if (r) entries.push({ type: 'item', view: r.view, label: r.nav!.label, icon: r.icon })
    }
  }

  // Ресурс с nav, которого нет в NAV_ORDER (новая верхнеуровневая запись), не теряем: в конец
  for (const r of resources) {
    if (r.nav && !r.nav.group && !NAV_ORDER.includes(r.view)) {
      entries.push({ type: 'item', view: r.view, label: r.nav.label, icon: r.icon })
    }
  }
  return entries
}

/** Группа с пунктами и вложенными группами; пустая (без пунктов и детей) не показывается */
function buildGroup(id: NavGroupId): NavGroup | null {
  const items: NavItem[] = [
    ...(NAV_GROUPS[id].views ?? []),
    ...resources
      .filter(r => r.nav?.group === id)
      .map(r => ({ view: r.view, label: r.nav!.label, icon: r.icon })),
  ]
  const children = (Object.keys(NAV_GROUPS) as NavGroupId[])
    .filter(child => NAV_GROUPS[child].parent === id)
    .map(buildGroup)
    .filter((g): g is NavGroup => g !== null)
  if (!items.length && !children.length) return null
  const meta = NAV_GROUPS[id]
  return { type: 'group', id, title: meta.title, icon: groupIcon(id), requires: meta.requires, items, children }
}

/** Иконки для групп бокового меню */
function groupIcon(id: NavGroupId): string {
  switch (id) {
    case 'workloads': return 'service'
    case 'config': return 'configmap'
    case 'network': return 'ingress'
    case 'storage': return 'pvc'
    case 'helm': return 'documentation'
    case 'access': return 'service-account'
    case 'argocd': return 'documentation'
    case 'argocd-settings': return 'documentation'
    default: return ''
  }
}
