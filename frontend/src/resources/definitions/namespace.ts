import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'

export const namespaceResource = defineResource<k8s.NamespaceInfo>({
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
    // get нет: панель показывает элемент списка (отдельного GetNamespaceDetail в Go нет)
    // remove намеренно нет: удаление namespace не поддерживалось и раньше
  },
})
