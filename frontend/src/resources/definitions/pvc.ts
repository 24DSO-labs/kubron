import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol, controlledByProp, managedByProp } from './common'
import { badgeClass, statusColor } from '../status'
import PVCMetricsSection from '../../components/resource/sections/PVCMetricsSection.vue'
import PVCSelectorSection from '../../components/resource/sections/PVCSelectorSection.vue'

  
import { expandAccessModes } from '../utils'

const MODES: Record<string, string> = {
  RWO: 'ReadWriteOnce',
  ROX: 'ReadOnlyMany',
  RWX: 'ReadWriteMany',
  RWOP: 'ReadWriteOncePod',
}
const accessModes = (p: any): string =>
  String(p.accessModes ?? '').split(',').filter(Boolean).map(m => MODES[m] ?? m).join(', ')

const size = (p: any): string => p.requested || p.capacity || ''
const scLink = (p: any) => (p.storageClass ? { id: 'storageclass', ns: '', name: p.storageClass } : null)

export const persistentVolumeClaim = defineResource<k8s.PVCInfo>({
  id: 'persistentvolumeclaim',
  kind: 'PersistentVolumeClaim',
  view: 'pvc',
  title: 'PersistentVolumeClaims',
  namespaced: true,
  nav: { group: 'storage', label: 'Persistent Volume Claims' },
  autoRefresh: true,

  // Как в списке-образце: Name, Namespace, Storage class, Size, Pods, Age, Status
  columns: [
    flatNameCol,
    { ...flatNsCol, link: (p: any) => p.namespace ? { id: 'namespace', ns: '', name: p.namespace } : null },
    { key: 'storageClass', label: 'Storage class', link: scLink },
    { key: 'size', label: 'Size', formatter: size },
    {
      key: 'pods',
      label: 'Pods',
      formatter: (p: any) => (p.pods ?? []).join(', ') || '—',
      link: (p: any) => (p.pods?.length === 1 ? { id: 'pod', ns: p.namespace, name: p.pods[0] } : null),
    },
    ageCol,
    { key: 'status', label: 'Status', colorFn: p => statusColor(p.status) },
  ],

  headSections: [PVCMetricsSection],

  // Name / Namespace / Age / Created выводит шапка панели; Labels и Annotations — общие секции
  properties: [
    { label: 'Finalizers', value: (p: any) => (p.finalizers ?? []).join(', ') },
    { label: 'Access Modes', value: (p: any) => expandAccessModes(p.accessModes) },
    { label: 'Storage Class Name', value: (p: any) => p.storageClass, link: scLink },
    { label: 'Storage', value: size },
    { label: 'Volume Mode', value: (p: any) => p.volumeMode },
    {
      label: 'Volume',
      value: (p: any) => p.volume,
      link: (p: any) => (p.volume ? { id: 'persistentvolume', ns: '', name: p.volume } : null),
    },
    {
      label: 'Pods',
      value: () => null,
      links: (p: any) => (p.pods ?? []).map((n: string) => ({ id: 'pod', ns: p.namespace, name: n })),
    },
    { label: 'Status', value: p => p.status, badge: p => badgeClass(p.status) },
    controlledByProp,
    managedByProp,
  ],
  sections: [PVCSelectorSection],

  api: {
    list: ns => callApp('GetPersistentVolumeClaims', ns),
    remove: (ns, name) => callApp('DeleteResource', 'persistentvolumeclaim', ns, name),
  },
})