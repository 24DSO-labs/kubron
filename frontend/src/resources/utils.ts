/** В списках встречаются оба формата: плоский (name) и k8s-подобный (metadata.name) */
export function itemName(item: any): string {
  return item?.name ?? item?.metadata?.name ?? ''
}

export function itemNs(item: any): string {
  return item?.namespace ?? item?.metadata?.namespace ?? ''
}

/**
 * Возраст из формата Go formatAge ("3d4h", "5h12m", "7m") в минуты: для сортировки.
 * null — не удалось разобрать («—», пусто).
 */
export function ageToMinutes(age?: string | null): number | null {
  const m = /^(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?$/.exec((age ?? '').trim())
  if (!m || !m[0]) return null
  return Number(m[1] ?? 0) * 1440 + Number(m[2] ?? 0) * 60 + Number(m[3] ?? 0)
}

/** Map labels/selector -> 'a=b, c=d' (в панели объект иначе показывается как [object Object]) */
export function formatMap(m?: Record<string, string> | null): string {
  return m ? Object.entries(m).map(([k, v]) => `${k}=${v}`).join(', ') : ''
}

/** Обычные (не init) контейнеры пода: их предлагаем в attach/shell/logs */
export function containersOf(item: any): { name: string }[] {
  return ((item?.containers ?? []) as { name: string; isInit?: boolean }[]).filter(c => !c.isInit)
}

export function matchesSearch(item: any, search: string): boolean {
  const s = search.trim().toLowerCase()
  if (!s) return true
  return itemName(item).toLowerCase().includes(s) || itemNs(item).toLowerCase().includes(s)
}

const PHASE_ORDER = ['Running', 'Pending', 'Succeeded', 'Failed', 'Unknown']

/** Pod Status из счётчиков фаз: "Running: 1" / "Running: 1, Pending: 2" */
export function podStatusSummary(item: any): string {
  const entries = Object.entries(item?.podStatuses ?? {}).filter(([, n]) => (n as number) > 0)
  if (!entries.length) return ''
  entries.sort((a, b) => {
    const ia = PHASE_ORDER.indexOf(a[0])
    const ib = PHASE_ORDER.indexOf(b[0])
    return (ia === -1 ? PHASE_ORDER.length : ia) - (ib === -1 ? PHASE_ORDER.length : ib)
  })
  return entries.map(([phase, n]) => `${phase}: ${n}`).join(', ')
}

export type PodOwnerKind = 'deployment' | 'replicaset' | 'daemonset' | 'statefulset' | 'job'

/**
 * Владелец подов для PromQL: kube_pod_owner{owner_kind, owner_name=~regex}
 * (regex в Prometheus всегда с якорями).
 * Deployment владеет ReplicaSet-ом "<dep>-<pod-template-hash>", поэтому для него ищем такие ReplicaSet.
 */
export function podOwner(kind: PodOwnerKind, name: string): { kind: string; regex: string } {
  const n = name.replace(/\./g, '[.]')
  switch (kind) {
    case 'deployment': return { kind: 'ReplicaSet', regex: `${n}-[a-z0-9]{8,10}` }
    case 'replicaset': return { kind: 'ReplicaSet', regex: n }
    case 'daemonset': return { kind: 'DaemonSet', regex: n }
    case 'statefulset': return { kind: 'StatefulSet', regex: n }
    case 'job': return { kind: 'Job', regex: n }
  }
}

const ACCESS_MODES: Record<string, string> = {
  RWO: 'ReadWriteOnce',
  ROX: 'ReadOnlyMany',
  RWX: 'ReadWriteMany',
  RWOP: 'ReadWriteOncePod',
}
/** "RWO,RWX" -> "ReadWriteOnce, ReadWriteMany" */
export function expandAccessModes(short?: string): string {
  return String(short ?? '').split(',').filter(Boolean).map(m => ACCESS_MODES[m] ?? m).join(', ')
}