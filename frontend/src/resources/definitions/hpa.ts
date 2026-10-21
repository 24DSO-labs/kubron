import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import ConditionsSection from '../../components/resource/sections/ConditionsSection.vue'
import HpaMetricsSection from '../../components/resource/sections/HpaMetricsSection.vue'

/** Kind цели автоскейлера -> id ресурса в приложении (для ссылки). Прочие kind ссылкой не делаем */
const TARGET_IDS: Record<string, string> = {
  Deployment: 'deployment',
  StatefulSet: 'statefulset',
  ReplicaSet: 'replicaset',
}

const scalingOk = (h: any) => (h.status ?? '').includes('ScalingActive')

export const horizontalPodAutoscaler = defineResource<k8s.HPAInfo>({
  id: 'horizontalpodautoscaler',
  kind: 'HPA',
  view: 'hpa',
  title: 'HorizontalPodAutoscalers',
  namespaced: true,
  nav: { group: 'config', label: 'HPA' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (h: any) => h.namespace ? { id: 'namespace', ns: '', name: h.namespace } : null },
    { key: 'metricsSummary', label: 'Metrics', formatter: (h: any) => h.metricsSummary || '—' },
    { key: 'minReplicas', label: 'Min Pods' },
    { key: 'maxReplicas', label: 'Max Pods' },
    { key: 'currentReplicas', label: 'Replicas' },
    { key: 'status', label: 'Status', formatter: (h: any) => h.status || '—', colorFn: (h: any) => (scalingOk(h) ? '#4ade80' : '#facc15') },
    ageCol,
  ],

  properties: [
    {
      label: 'Reference',
      value: (h: any) => `${h.referenceKind}/${h.referenceName}`,
      link: (h: any) => {
        const id = TARGET_IDS[h.referenceKind]
        return id && h.referenceName ? { id, ns: h.namespace, name: h.referenceName } : null
      },
    },
    { label: 'Min Replicas', value: (h: any) => h.minReplicas },
    { label: 'Max Replicas', value: (h: any) => h.maxReplicas },
    { label: 'Current Replicas', value: (h: any) => h.currentReplicas },
    { label: 'Desired Replicas', value: (h: any) => h.desiredReplicas },
    { label: 'Last Scale', value: (h: any) => h.lastScaleTime },
    { label: 'Finalizers', value: (h: any) => (h.finalizers ?? []).join(', ') },
    controlledByProp,
    managedByProp,
  ],
  sections: [ConditionsSection, HpaMetricsSection],

  api: {
    list: ns => callApp('GetHPA', ns),
    remove: (ns, name) => callApp('DeleteResource', 'horizontalpodautoscaler', ns, name),
  },
})
