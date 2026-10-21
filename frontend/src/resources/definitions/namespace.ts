import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import { badgeClass, statusColor } from '../status'
import NamespaceMetricsSection from '../../components/resource/sections/NamespaceMetricsSection.vue'

export const namespaceResource = defineResource<k8s.NamespaceInfo>({
  id: 'namespace',
  kind: 'Namespace',
  view: 'namespaces',
  title: 'Namespaces',
  // icon: 'namespace',
  namespaced: false,
  nav: { label: 'Namespaces' },




  autoRefresh: true,
  columns: [
    flatNameCol,
    { key: 'status', label: 'Status', colorFn: n => statusColor(n.status) },
    ageCol,
  ],
  headSections: [NamespaceMetricsSection],
  properties: [{ label: 'Status', value: n => n.status, badge: n => badgeClass(n.status) }, controlledByProp, managedByProp],

  api: {
    list: () => callApp('GetNamespaces'),
    // get нет: панель показывает элемент списка (отдельного GetNamespaceDetail в Go нет)
    // remove намеренно нет: удаление namespace не поддерживалось и раньше
  },
})
