import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'

export const resourceQuota = defineResource<k8s.ResourceQuotaInfo>({
  id: 'resourcequota',
  kind: 'ResourceQuota',
  view: 'resourcequotas',
  title: 'ResourceQuotas',
  namespaced: true,
  nav: { group: 'config', label: 'Resource Quotas' },
  
  autoRefresh: true,
  // icon: 'configmap',

  columns: [
    flatNameCol,
    { ...flatNsCol, link: (r: any) => r.namespace ? { id: 'namespace', ns: '', name: r.namespace } : null },
    { key: 'cpuUsed', label: 'CPU Used' },
    { key: 'memLimit', label: 'Memory Limit' },
    { key: 'memUsed', label: 'Memory Used' },
    ageCol,
  ],

  properties: [
    { label: 'CPU Limit', value: q => q.cpuLimit },
    { label: 'CPU Used', value: q => q.cpuUsed },
    { label: 'Memory Limit', value: q => q.memLimit },
    { label: 'Memory Used', value: q => q.memUsed },
  ],

  api: {
    list: ns => callApp('GetResourceQuotas', ns),
    remove: (ns, name) => callApp('DeleteResourceQuota', ns, name),
  },
})
