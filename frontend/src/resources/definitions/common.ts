import type { ColumnDef, PropertyDef } from '../types'
import { ageToMinutes } from '../utils'

// Элементы Role/RoleBinding/... приходят в k8s-формате (metadata.name)
export const nameCol: ColumnDef = { key: 'metadata.name', label: 'Name', className: 'resource-name' }
export const nsCol: ColumnDef = { key: 'metadata.namespace', label: 'Namespace', className: 'ns-label' }

// Остальные приходят плоскими (name, namespace)
export const flatNameCol: ColumnDef = { key: 'name', label: 'Name', className: 'resource-name' }
export const flatNsCol: ColumnDef = { key: 'namespace', label: 'Namespace', className: 'ns-label' }

export const ageCol: ColumnDef = { key: 'age', label: 'Age', sortValue: (r: any) => ageToMinutes(r.age) }

// ─── Controlled By / Managed by ───
// Бэкенд отдаёт оба поля строкой "Kind/name" (controlledBy — владелец из ownerReferences,
// managedBy — Argo CD / Helm, см. internal/k8s/configmeta.go)

/** Kind -> id ресурса в приложении. Для остальных (CRD: ExternalSecret, Certificate ...) ссылки нет */
const OWNER_IDS: Record<string, string> = {
  ReplicaSet: 'replicaset',
  StatefulSet: 'statefulset',
  DaemonSet: 'daemonset',
  Job: 'job',
  CronJob: 'cronjob',
  Deployment: 'deployment',
  Node: 'node',
  Service: 'service',
  Ingress: 'ingress',
  ConfigMap: 'configmap',
  Secret: 'secret',
}

const splitRef = (ref?: string): [string, string] => {
  if (!ref) return ['', '']
  const i = ref.indexOf('/')
  return i < 0 ? [ref, ''] : [ref.slice(0, i), ref.slice(i + 1)]
}

const metaOf = (r: any) => ({
  labels: (r?.labels ?? r?.metadata?.labels ?? {}) as Record<string, string>,
  annotations: (r?.annotations ?? r?.metadata?.annotations ?? {}) as Record<string, string>,
})

/**
 * Владелец: готовое поле controlledBy от бэкенда, иначе из metadata.ownerReferences
 * (Role, ClusterRole и другие ресурсы приходят в k8s-формате). Берётся controller-владелец.
 */
function controlledByOf(r: any): string {
  if (r?.controlledBy) return r.controlledBy
  const refs: any[] = r?.metadata?.ownerReferences ?? r?.ownerReferences ?? []
  const ref = refs.find(x => x.controller) ?? refs[0]
  return ref ? `${ref.kind}/${ref.name}` : ''
}

/**
 * Кто управляет объектом. Те же правила, что managedByOf в internal/k8s/configmeta.go:
 * Argo CD (tracking-id или label instance) и Helm. Нужны на фронтенде для ресурсов,
 * которые бэкенд отдаёт без поля managedBy.
 */
function managedByOf(r: any): string {
  if (r?.managedBy) return r.managedBy
  const { labels, annotations } = metaOf(r)
  const trackingApp = annotations['argocd.argoproj.io/tracking-id']?.split(':')[0]
  if (trackingApp) return `Application/${trackingApp}`
  if (labels['argocd.argoproj.io/instance']) return `Application/${labels['argocd.argoproj.io/instance']}`
  if (labels['app.kubernetes.io/managed-by'] === 'Helm' && annotations['meta.helm.sh/release-name']) {
    return `Helm/${annotations['meta.helm.sh/release-name']}`
  }
  return ''
}

/** Свойство «Controlled By»: "Kind name"; имя — ссылка, если kind известен приложению. Пустое скрывается */
export const controlledByProp: PropertyDef = {
  label: 'Controlled By',
  hideEmpty: true,
  value: (r: any) => {
    const [kind, name] = splitRef(controlledByOf(r))
    if (!kind) return null
    return OWNER_IDS[kind] && name ? `${kind} ` : `${kind} ${name}`.trim()
  },
  linkLabel: (r: any) => splitRef(controlledByOf(r))[1] || null,
  link: (r: any) => {
    const [kind, name] = splitRef(controlledByOf(r))
    const id = OWNER_IDS[kind]
    return id && name ? { id, ns: r.namespace ?? r.metadata?.namespace ?? '', name } : null
  },
}

/** Свойство «Managed by»: "Application/name" (Argo CD) или "Helm/release". Ссылок нет: эти ресурсы приложение не открывает */
export const managedByProp: PropertyDef = {
  label: 'Managed by',
  hideEmpty: true,
  value: (r: any) => managedByOf(r) || null,
}
