/** В списках встречаются оба формата: плоский (name) и k8s-подобный (metadata.name) */
export function itemName(item: any): string {
  return item?.name ?? item?.metadata?.name ?? ''
}

export function itemNs(item: any): string {
  return item?.namespace ?? item?.metadata?.namespace ?? ''
}

export function matchesSearch(item: any, search: string): boolean {
  const s = search.trim().toLowerCase()
  if (!s) return true
  return itemName(item).toLowerCase().includes(s) || itemNs(item).toLowerCase().includes(s)
}
