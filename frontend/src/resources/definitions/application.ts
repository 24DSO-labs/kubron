import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNsCol, ageCol } from './common'
import { badgeClass, statusColor } from '../status'
import { applicationReplicas, urlsText } from '../applications'

import ApplicationMetricsSection from '../../components/resource/sections/ApplicationMetricsSection.vue'
import ApplicationPodsSection from '../../components/resource/sections/ApplicationPodsSection.vue'
import ApplicationResourcesSection from '../../components/resource/sections/ApplicationResourcesSection.vue'

/** «ready/desired» по подам всех workload-ов приложения */
const podsText = (a: any) => {
  const { ready, desired } = applicationReplicas(a)
  return `${ready}/${desired}`
}

/**
 * Applications (как в Lens): workload-ы, сгруппированные по recommended labels
 * app.kubernetes.io/instance + app.kubernetes.io/name. Это вид поверх Deployment/StatefulSet/DaemonSet,
 * а не отдельный k8s-ресурс: без YAML и удаления.
 */
export const application = defineResource<k8s.ApplicationInfo>({
  // id не 'application': он занят Argo CD Application (definitions/argocd/application.ts),
  // а реестр резолвит определения по id (registry.ts) — дубликат молча перекрыл бы Argo-вкладки
  id: 'app-instance',
  kind: 'ApplicationInstance',
  view: 'applications',
  title: 'Applications',
  namespaced: true,
  nav: { label: 'Applications' },
  icon: 'dependencies',

  autoRefresh: true,
  editable: false,

  columns: [
    { key: 'instance', label: 'Instance', className: 'resource-name' },
    { key: 'application', label: 'Application' },
    flatNsCol,
    { key: 'version', label: 'Version' },
    { key: 'managedBy', label: 'Managed By' },
    {
      key: 'pods',
      label: 'Pods',
      formatter: podsText,
      sortValue: (a: any) => applicationReplicas(a).ready,
      colorFn: (a: any) => {
        const { ready, desired } = applicationReplicas(a)
        return ready >= desired ? '#4ade80' : '#facc15'
      },
    },
    ageCol,
    { key: 'status', label: 'Status', colorFn: (a: any) => statusColor(a.status) },
  ],

  headSections: [ApplicationMetricsSection],
  properties: [
    { label: 'Status', value: a => a.status, badge: a => badgeClass(a.status) },
    { label: 'Application', value: a => a.application },
    { label: 'Version', value: a => a.version },
    { label: 'Managed By', value: a => a.managedBy },
    { label: 'Component', value: a => a.component, hideEmpty: true },
    { label: 'Part Of', value: a => a.partOf, hideEmpty: true },
    { label: 'Internal URLs', multiline: true, value: a => urlsText(a.internalUrls) },
    { label: 'Exposed URLs', multiline: true, value: a => urlsText(a.exposedUrls) },
    { label: 'Name', value: a => a.name },
    {
      label: 'Namespace',
      value: a => a.namespace,
      link: a => (a.namespace ? { id: 'namespace', ns: '', name: a.namespace } : null),
    },
  ],
  sections: [ApplicationPodsSection, ApplicationResourcesSection],

  api: {
    list: ns => callApp('GetApplications', ns),
  },
})
