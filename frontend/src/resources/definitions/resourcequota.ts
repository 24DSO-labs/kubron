import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import QuotaUsageSection from '../../components/resource/sections/QuotaUsageSection.vue'

export const resourceQuota = defineResource<k8s.ResourceQuotaInfo>({
  id: 'resourcequota',
  kind: 'ResourceQuota',
  view: 'resourcequotas',
  title: 'ResourceQuotas',
  namespaced: true,
  nav: { group: 'config', label: 'Resource Quotas' },

  autoRefresh: true,

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (r: any) => r.namespace ? { id: 'namespace', ns: '', name: r.namespace } : null },
    ageCol,
  ],

  properties: [
    { label: 'Finalizers', value: (q: any) => (q.finalizers ?? []).join(', ') },
  ],
  sections: [QuotaUsageSection],

  api: {
    list: ns => callApp('GetResourceQuotas', ns),
    remove: (ns, name) => callApp('DeleteResource', 'resourcequota', ns, name),
  },
})
