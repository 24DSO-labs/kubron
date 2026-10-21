import type { ColumnDef, PropertyDef } from '../types'

// Элементы Role/RoleBinding/... приходят в k8s-формате (metadata.name)
export const nameCol: ColumnDef = { key: 'metadata.name', label: 'Name', className: 'resource-name' }
export const nsCol: ColumnDef = { key: 'metadata.namespace', label: 'Namespace', className: 'ns-label' }

// Остальные приходят плоскими (name, namespace)
export const flatNameCol: ColumnDef = { key: 'name', label: 'Name', className: 'resource-name' }
export const flatNsCol: ColumnDef = { key: 'namespace', label: 'Namespace', className: 'ns-label' }

export const ageCol: ColumnDef = { key: 'age', label: 'Age' }

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

/** Свойство «Controlled By»: "Kind name"; имя — ссылка, если kind известен приложению */
export const controlledByProp: PropertyDef = {
  label: 'Controlled By',
  value: (r: any) => {
    const [kind, name] = splitRef(r.controlledBy)
    if (!kind) return null
    return OWNER_IDS[kind] && name ? `${kind} ` : `${kind} ${name}`.trim()
  },
  linkLabel: (r: any) => splitRef(r.controlledBy)[1] || null,
  link: (r: any) => {
    const [kind, name] = splitRef(r.controlledBy)
    const id = OWNER_IDS[kind]
    return id && name ? { id, ns: r.namespace ?? '', name } : null
  },
}

/** Свойство «Managed by»: "Application/name" (Argo CD) или "Helm/release". Ссылок нет: эти ресурсы приложение не открывает */
export const managedByProp: PropertyDef = {
  label: 'Managed by',
  value: (r: any) => r.managedBy,
}
