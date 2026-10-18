/** В списках встречаются оба формата: плоский (name) и k8s-подобный (metadata.name) */
export function itemName(item: any): string {
  return item?.name ?? item?.metadata?.name ?? ''
}

export function itemNs(item: any): string {
  return item?.namespace ?? item?.metadata?.namespace ?? ''
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

/** Regex имён подов ресурса для PromQL (pod=~"..."; в Prometheus regex всегда с якорями) */
export function podNameRegex(kind: PodOwnerKind, name: string): string {
  const n = name.replace(/\./g, '[.]')
  switch (kind) {
    case 'deployment': return `${n}-[a-z0-9]{8,10}-[a-z0-9]{5}` // <dep>-<hash ReplicaSet>-<5>
    case 'statefulset': return `${n}-[0-9]+`
    case 'job': return `${n}-([0-9]+-)?[a-z0-9]{5}` // у Indexed-Job в имени есть индекс
    default: return `${n}-[a-z0-9]{5}` // daemonset, replicaset
  }
}