import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const namespaceResource = defineResource({
  id: 'namespace',
  kind: 'Namespace',
  view: 'namespaces',
  title: 'Namespaces',
  icon: '📂',
  namespaced: false,
  nav: { label: 'Namespaces' },

  columns: [
    flatNameCol,
    { key: 'status', label: 'Status', colorFn: n => statusColor(n.status) },
    ageCol,
  ],

  properties: [{ label: 'Status', value: n => n.status, badge: n => badgeClass(n.status) }],

  api: {
    list: () => callApp('GetNamespaces'),
    // GetNamespaceDetail необязателен: без него панель показывает элемент списка
    get: (_ns, name) => callApp('GetNamespaceDetail', name),
    // remove намеренно нет: удаление namespace не поддерживалось и раньше
  },
})
