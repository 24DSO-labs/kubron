import { callApp } from './wails'
import type { k8s } from '../../wailsjs/go/models'

/**
 * Содержимое ревизии Helm неизменяемо, поэтому кешируется по ns/name/revision.
 * Кеш ограничен (LRU) и сбрасывается при смене кластера (см. App.vue): в ключе кластера нет.
 */
const MAX = 20
const cache = new Map<string, Promise<k8s.HelmReleaseContent>>()

export function loadHelmContent(ns: string, name: string, revision: number): Promise<k8s.HelmReleaseContent> {
  // revision <= 0 («последняя») не кешируем: результат меняется со временем
  if (revision <= 0) return callApp('GetHelmReleaseContent', ns, name, revision)

  const key = `${ns}/${name}/${revision}`
  let p = cache.get(key)
  if (p) {
    cache.delete(key) // обновляем порядок LRU
    cache.set(key, p)
    return p
  }
  p = callApp('GetHelmReleaseContent', ns, name, revision)
  p.catch(() => cache.delete(key)) // ошибку (например, нет прав) не запоминаем
  cache.set(key, p)
  if (cache.size > MAX) cache.delete(cache.keys().next().value!)
  return p
}

export const clearHelmContentCache = () => cache.clear()