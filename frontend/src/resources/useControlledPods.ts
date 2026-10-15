import { ref, computed, watch } from 'vue'
import { useK8s } from '../composables/useK8s'
import { itemName, itemNs } from './utils'

const k8s = useK8s()

/** Имя владельца из controlledBy: "ReplicaSet/my-app-5d8c" -> "my-app-5d8c" */
const ownerName = (controlledBy: string) => controlledBy.split(/[\s/:]+/).filter(Boolean).pop() ?? ''

/** Принадлежит ли pod ресурсу: по владельцу, иначе по label selector ("k=v,k2=v2") */
function belongs(pod: any, name: string, selector?: string): boolean {
  if (pod.controlledBy) {
    const owner = ownerName(String(pod.controlledBy))
    // Deployment владеет ReplicaSet-ом "<name>-<hash>"; hash без дефисов, поэтому "my-app-db-xyz" не подходит под "my-app"
    if (owner === name || (owner.startsWith(name + '-') && !owner.slice(name.length + 1).includes('-'))) return true
  }
  if (pod.labels && selector) {
    const parts = selector.split(',').map(s => s.trim()).filter(Boolean)
    return parts.length > 0 && parts.every(part => {
      const [key, value] = part.split('=').map(s => s.trim())
      return pod.labels[key] === value
    })
  }
  return false
}

/**
 * Поды, которыми управляет ресурс (Deployment, DaemonSet, ReplicaSet).
 * getResource() может вернуть null: тогда ничего не грузится.
 */
export function useControlledPods(getResource: () => any) {
  const pods = ref<any[]>([])
  const loading = ref(false)

  async function load() {
    const r = getResource()
    if (!r) { pods.value = []; return }
    const ns = itemNs(r)
    const name = itemName(r)
    loading.value = true
    try {
      const all = ((await k8s.getPods(ns)) as any[]) ?? []
      pods.value = all.filter(p => p.namespace === ns && belongs(p, name, r.selector))
    } catch {
      pods.value = []
    } finally {
      loading.value = false
    }
  }

  watch(getResource, load, { immediate: true })

  /** Метрики по имени пода (заполняет k8s.loadPodMetrics) */
  const metrics = computed(() => {
    const map: Record<string, any> = {}
    for (const m of k8s.podMetrics.value as any[]) map[m.name] = m
    return map
  })

  return { pods, loading, metrics, reload: load, formatCpu: k8s.formatCpu, formatMemory: k8s.formatMemory, loadMetrics: k8s.loadPodMetrics }
}
