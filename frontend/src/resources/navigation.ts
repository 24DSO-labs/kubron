import { resources } from './registry'
import type { NavGroupId } from './types'

/** Заголовки групп и начальное состояние (свёрнута/развёрнута) */
export const NAV_GROUPS: Record<NavGroupId, { title: string; expanded: boolean }> = {
  workloads: { title: 'Workloads', expanded: true },
  config: { title: 'Config', expanded: false },
  network: { title: 'Network', expanded: true },
  storage: { title: 'Storage', expanded: false },
  helm: { title: 'Helm', expanded: false },
  access: { title: 'Access Control', expanded: false },
}

/**
 * Порядок бокового меню между Overview и Settings.
 * Элемент — id группы или view ресурса верхнего уровня (у него в `nav` нет group).
 */
const NAV_ORDER: string[] = [
  'nodes', 'events',
  'workloads', 'config', 'network', 'storage',
  'namespaces',
  'helm', 'access',
]

export interface NavItem { view: string; label: string; icon?: string }
export type NavEntry =
  | { type: 'item'; view: string; label: string; icon?: string }
  | { type: 'group'; id: NavGroupId; title: string; icon?: string; items: NavItem[] }

const isGroup = (id: string): id is NavGroupId => id in NAV_GROUPS

/** Меню строится из `nav` в определениях ресурсов: новый ресурс появляется в нём без правок Sidebar */
export function buildNav(): NavEntry[] {
  const entries: NavEntry[] = []

  for (const id of NAV_ORDER) {
    if (isGroup(id)) {
      const items = resources
        .filter(r => r.nav?.group === id)
        .map(r => ({ view: r.view, label: r.nav!.label, icon: r.icon }))
      if (items.length) entries.push({ type: 'group', id, title: NAV_GROUPS[id].title, icon: groupIcon(id), items })
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

/** Иконки для групп бокового меню */
function groupIcon(id: NavGroupId): string {
  switch (id) {
    case 'workloads': return 'service'
    case 'config': return 'configmap'
    case 'network': return 'ingress'
    case 'storage': return 'pvc'
    case 'helm': return 'documentation'
    case 'access': return 'service-account'
    default: return ''
  }
}
