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
