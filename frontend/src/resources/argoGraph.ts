/**
 * Раскладка графа ресурсов Argo CD (без Vue и Wails): слои слева направо по самому длинному пути от приложения,
 * порядок внутри слоя по среднему положению родителей, схлопывание больших групп подов.
 */
import type { ArgoEdgeType, ArgoGraph, ArgoGraphEdge, ArgoGraphNode } from '../types/argocd'

export const NODE_W = 190
export const NODE_H = 46
export const GAP_X = 70
export const GAP_Y = 14
/** Больше стольких одинаковых «листовых» подов под одним родителем схлопываются в одну вершину */
export const COLLAPSE_AT = 6
/** Больше стольких одиночных ресурсов одного Kind (ConfigMap, ClusterRole ...) схлопываются в одну вершину */
export const KIND_COLLAPSE_AT = 4
export const ROOT_ID = '__app__'
export const GROUP_PREFIX = 'group:'
/** id группы по Kind: `group:kind:ConfigMap` (группы подов — `group:<id родителей>`) */
export const KIND_GROUP_PREFIX = GROUP_PREFIX + 'kind:'

export interface PlacedNode {
  id: string
  kind: string
  name: string
  namespace: string
  status: string
  health: string
  message: string
  managed: boolean
  hook: boolean
  x: number
  y: number
  w: number
  h: number
  isRoot: boolean
  /** для схлопнутой группы: сколько подов и их id */
  count: number
  members: string[]
}

export interface PlacedEdge {
  id: string
  from: string
  to: string
  type: ArgoEdgeType
  d: string
}

export interface GraphLayout {
  nodes: PlacedNode[]
  edges: PlacedEdge[]
  width: number
  height: number
}

const SEVERITY: Record<string, number> = { Degraded: 5, Missing: 5, Progressing: 4, Suspended: 3, Unknown: 2, Healthy: 1 }

/** Более «тяжёлый» из двух health-статусов (для сводки схлопнутой группы) */
export function worstHealth(a: string, b: string): string {
  return (SEVERITY[b] ?? 0) > (SEVERITY[a] ?? 0) ? b : a
}

type GNode = ArgoGraphNode & { count?: number; members?: string[] }

const edgeKey = (e: ArgoGraphEdge) => `${e.from}|${e.to}|${e.type}`

/** Родители каждой вершины */
function incomingOf(edges: ArgoGraphEdge[]): Map<string, ArgoGraphEdge[]> {
  const m = new Map<string, ArgoGraphEdge[]>()
  for (const e of edges) {
    const list = m.get(e.to)
    if (list) list.push(e)
    else m.set(e.to, [e])
  }
  return m
}

/**
 * Оставляет вершины, подходящие под предикат, и всех их предков: так фильтр по Kind/Health
 * не рвёт связность (под остаётся вместе с ReplicaSet и Deployment).
 */
export function filterGraph(g: ArgoGraph, keep: (n: ArgoGraphNode) => boolean): ArgoGraph {
  const incoming = incomingOf(g.edges)
  const byId = new Map(g.nodes.map(n => [n.id, n]))
  const kept = new Set<string>()
  const stack: string[] = []
  for (const n of g.nodes) if (keep(n)) stack.push(n.id)
  while (stack.length) {
    const id = stack.pop()!
    if (kept.has(id) || !byId.has(id)) continue
    kept.add(id)
    for (const e of incoming.get(id) ?? []) stack.push(e.from)
  }
  return {
    ...g,
    nodes: g.nodes.filter(n => kept.has(n.id)),
    edges: g.edges.filter(e => kept.has(e.from) && kept.has(e.to)),
  }
}

/**
 * Схлопывает «листовые» поды с одним и тем же набором родителей, если их больше COLLAPSE_AT.
 * Развёрнутые группы (id в expanded) остаются как есть.
 */
export function collapseLeaves(g: ArgoGraph, expanded: ReadonlySet<string>): { nodes: GNode[]; edges: ArgoGraphEdge[] } {
  const hasOutgoing = new Set(g.edges.map(e => e.from))
  const incoming = incomingOf(g.edges)

  const groups = new Map<string, GNode[]>()
  for (const n of g.nodes) {
    if (n.kind !== 'Pod' || hasOutgoing.has(n.id)) continue
    const parents = [...new Set((incoming.get(n.id) ?? []).map(e => e.from))].sort().join('|')
    if (!parents) continue // под без родителей остаётся отдельной вершиной
    const list = groups.get(parents)
    if (list) list.push(n)
    else groups.set(parents, [n])
  }

  const remap = new Map<string, string>()
  const synthetic: GNode[] = []
  for (const [key, members] of groups) {
    const id = GROUP_PREFIX + key
    if (members.length <= COLLAPSE_AT || expanded.has(id)) continue
    let health = ''
    for (const m of members) health = worstHealth(health, m.health)
    synthetic.push({
      id, group: '', kind: 'Pod', namespace: members[0].namespace, name: `${members.length} pods`,
      status: '', health, message: '', hook: false, managed: false,
      count: members.length, members: members.map(m => m.id).sort(),
    })
    for (const m of members) remap.set(m.id, id)
  }
  if (!remap.size) return { nodes: g.nodes, edges: g.edges }

  const seen = new Set<string>()
  const edges: ArgoGraphEdge[] = []
  for (const e of g.edges) {
    const r: ArgoGraphEdge = { ...e, from: remap.get(e.from) ?? e.from, to: remap.get(e.to) ?? e.to }
    const k = edgeKey(r)
    if (!seen.has(k)) {
      seen.add(k)
      edges.push(r)
    }
  }
  return { nodes: [...g.nodes.filter(n => !remap.has(n.id)), ...synthetic], edges }
}

/** Вершина, которую нельзя прятать в группу: ей нужно внимание (проблемный health или расхождение с git) */
export function needsAttention(n: Pick<ArgoGraphNode, 'status' | 'health'>): boolean {
  return n.status === 'OutOfSync' || (!!n.health && n.health !== 'Healthy' && n.health !== 'Suspended')
}

/**
 * Схлопывает «одиночные» ресурсы (без входящих и исходящих рёбер: ClusterRole, ConfigMap, CRD ...)
 * по Kind, если их больше KIND_COLLAPSE_AT. Проблемные ресурсы остаются отдельными вершинами,
 * чтобы их не пришлось искать внутри группы. Развёрнутые группы (id в expanded) не схлопываются.
 */
export function collapseKinds(
  nodes: GNode[], edges: ArgoGraphEdge[], expanded: ReadonlySet<string>,
): { nodes: GNode[]; edges: ArgoGraphEdge[] } {
  const connected = new Set<string>()
  for (const e of edges) {
    connected.add(e.from)
    connected.add(e.to)
  }

  const byKind = new Map<string, GNode[]>()
  for (const n of nodes) {
    if (n.id.startsWith(GROUP_PREFIX) || connected.has(n.id) || needsAttention(n)) continue
    const list = byKind.get(n.kind)
    if (list) list.push(n)
    else byKind.set(n.kind, [n])
  }

  const grouped = new Set<string>()
  const synthetic: GNode[] = []
  for (const [kind, members] of byKind) {
    const id = KIND_GROUP_PREFIX + kind
    if (members.length <= KIND_COLLAPSE_AT || expanded.has(id)) continue
    let health = ''
    for (const m of members) health = worstHealth(health, m.health)
    synthetic.push({
      id, group: '', kind, namespace: '', name: `${members.length} × ${kind}`,
      status: '', health, message: '', hook: false, managed: false,
      count: members.length, members: members.map(m => m.id).sort(),
    })
    for (const m of members) grouped.add(m.id)
  }
  if (!grouped.size) return { nodes, edges }
  return { nodes: [...nodes.filter(n => !grouped.has(n.id)), ...synthetic], edges }
}

/** Раскладывает граф; приложение — виртуальный корень слева, к нему подвешены вершины без входящих рёбер */
export function layoutGraph(g: ArgoGraph, appName: string, expanded: ReadonlySet<string> = new Set()): GraphLayout {
  const pods = collapseLeaves(g, expanded)
  const { nodes: gnodes, edges: gedges } = collapseKinds(pods.nodes, pods.edges, expanded)

  const root: GNode = {
    id: ROOT_ID, group: '', kind: 'Application', namespace: '', name: appName, status: '', health: '',
    message: '', hook: false, managed: true,
  }
  const hasIncoming = new Set(gedges.map(e => e.to))
  const edges: ArgoGraphEdge[] = [
    ...gedges,
    ...gnodes.filter(n => !hasIncoming.has(n.id)).map(n => ({ from: ROOT_ID, to: n.id, type: 'owns' as ArgoEdgeType })),
  ]
  const nodes: GNode[] = [root, ...gnodes]

  // Слой = самый длинный путь от корня; защита от циклов на случай некорректных данных
  const incoming = incomingOf(edges)
  const depth = new Map<string, number>()
  const visiting = new Set<string>()
  const depthOf = (id: string): number => {
    const known = depth.get(id)
    if (known !== undefined) return known
    if (visiting.has(id)) return 0
    visiting.add(id)
    let d = 0
    for (const e of incoming.get(id) ?? []) d = Math.max(d, depthOf(e.from) + 1)
    visiting.delete(id)
    depth.set(id, d)
    return d
  }
  for (const n of nodes) depthOf(n.id)

  const layers: GNode[][] = []
  for (const n of nodes) {
    const d = depth.get(n.id) ?? 0
    ;(layers[d] ??= []).push(n)
  }

  // Порядок внутри слоя: по среднему номеру строки родителей, затем по виду и имени
  const row = new Map<string, number>()
  layers.forEach((layer, li) => {
    if (li > 0) {
      const bary = (n: GNode) => {
        const ps = (incoming.get(n.id) ?? []).map(e => row.get(e.from)).filter((v): v is number => v !== undefined)
        return ps.length ? ps.reduce((a, b) => a + b, 0) / ps.length : Number.MAX_SAFE_INTEGER
      }
      layer.sort((a, b) =>
        bary(a) - bary(b) || a.kind.localeCompare(b.kind) || a.name.localeCompare(b.name))
    }
    layer.forEach((n, i) => row.set(n.id, i))
  })

  const placed = new Map<string, PlacedNode>()
  layers.forEach((layer, li) => {
    layer.forEach((n, i) => {
      placed.set(n.id, {
        id: n.id, kind: n.kind, name: n.name, namespace: n.namespace, status: n.status, health: n.health,
        message: n.message, managed: n.managed, hook: n.hook,
        x: li * (NODE_W + GAP_X), y: i * (NODE_H + GAP_Y), w: NODE_W, h: NODE_H,
        isRoot: n.id === ROOT_ID, count: n.count ?? 1, members: n.members ?? [],
      })
    })
  })

  const placedEdges: PlacedEdge[] = []
  for (const e of edges) {
    const a = placed.get(e.from)
    const b = placed.get(e.to)
    if (!a || !b) continue
    const x1 = a.x + a.w
    const y1 = a.y + a.h / 2
    const x2 = b.x
    const y2 = b.y + b.h / 2
    const dx = Math.max(20, (x2 - x1) / 2)
    placedEdges.push({
      id: edgeKey(e), from: e.from, to: e.to, type: e.type,
      d: `M${x1},${y1} C${x1 + dx},${y1} ${x2 - dx},${y2} ${x2},${y2}`,
    })
  }

  const all = [...placed.values()]
  return {
    nodes: all,
    edges: placedEdges,
    width: all.reduce((m, n) => Math.max(m, n.x + n.w), 0),
    height: all.reduce((m, n) => Math.max(m, n.y + n.h), 0),
  }
}
