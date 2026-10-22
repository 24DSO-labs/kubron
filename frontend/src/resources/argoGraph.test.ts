import { describe, expect, it } from 'vitest'
import type { ArgoEdgeType, ArgoGraph, ArgoGraphNode } from '../types/argocd'
import {
  COLLAPSE_AT, GROUP_PREFIX, KIND_COLLAPSE_AT, KIND_GROUP_PREFIX, NODE_H, NODE_W, ROOT_ID,
  collapseKinds, collapseLeaves, filterGraph, layoutGraph, needsAttention, worstHealth,
} from './argoGraph'

const node = (kind: string, name: string, p: Partial<ArgoGraphNode> = {}): ArgoGraphNode => {
  const group = p.group ?? ''
  const namespace = p.namespace ?? 'prod'
  return {
    id: `${group}/${kind}/${namespace}/${name}`, group, kind, namespace, name,
    status: '', health: '', message: '', hook: false, managed: true, ...p,
  }
}
const edge = (from: ArgoGraphNode, to: ArgoGraphNode, type: ArgoEdgeType = 'owns') => ({ from: from.id, to: to.id, type })
const graph = (nodes: ArgoGraphNode[], edges: ReturnType<typeof edge>[]): ArgoGraph =>
  ({ nodes, edges, truncated: false, warnings: [] })

const dep = node('Deployment', 'web', { group: 'apps' })
const rs = node('ReplicaSet', 'web-abc', { group: 'apps', managed: false })
const pod = (i: number, health = 'Healthy') => node('Pod', `web-abc-${i}`, { managed: false, health })

describe('layoutGraph', () => {
  it('ставит приложение корнем слева, цепочку владения по слоям вправо', () => {
    const p = pod(1)
    const l = layoutGraph(graph([dep, rs, p], [edge(dep, rs), edge(rs, p)]), 'my-app')
    const x = (id: string) => l.nodes.find(n => n.id === id)!.x

    expect(x(ROOT_ID)).toBe(0)
    expect(x(dep.id)).toBe(NODE_W + 70)
    expect(x(rs.id)).toBeGreaterThan(x(dep.id))
    expect(x(p.id)).toBeGreaterThan(x(rs.id))
    expect(l.nodes.find(n => n.isRoot)?.name).toBe('my-app')
    expect(l.edges.every(e => e.d.startsWith('M'))).toBe(true)
    // Корень соединён только с вершинами без входящих рёбер
    expect(l.edges.filter(e => e.from === ROOT_ID).map(e => e.to)).toEqual([dep.id])
  })

  it('слой вершины определяется самым длинным путём (под глубже и Ingress-ветки, и владельца)', () => {
    const ing = node('Ingress', 'web', { group: 'networking.k8s.io' })
    const svc = node('Service', 'web')
    const p = pod(1)
    const l = layoutGraph(
      graph([dep, rs, p, ing, svc], [edge(dep, rs), edge(rs, p), edge(ing, svc, 'routes'), edge(svc, p, 'selects')]),
      'app',
    )
    const x = (id: string) => l.nodes.find(n => n.id === id)!.x
    // Обе цепочки (Deployment → ReplicaSet и Ingress → Service) ведут к поду: он правее обоих своих родителей
    expect(x(p.id)).toBeGreaterThan(x(rs.id))
    expect(x(p.id)).toBeGreaterThan(x(svc.id))
    expect(l.edges.some(e => e.type === 'selects')).toBe(true)
    expect(l.edges.some(e => e.type === 'routes')).toBe(true)
  })

  it('вершины одного слоя не пересекаются по вертикали', () => {
    const a = node('ConfigMap', 'a')
    const b = node('ConfigMap', 'b')
    const l = layoutGraph(graph([a, b], []), 'app')
    const ys = l.nodes.filter(n => n.kind === 'ConfigMap').map(n => n.y).sort((p, q) => p - q)
    expect(ys[1] - ys[0]).toBeGreaterThanOrEqual(NODE_H)
  })

  it('не зависает на цикле в данных', () => {
    const a = node('Job', 'a')
    const b = node('Job', 'b')
    const l = layoutGraph(graph([a, b], [edge(a, b), edge(b, a)]), 'app')
    expect(l.nodes).toHaveLength(3) // корень + две вершины
  })

  it('пустой граф — только корень', () => {
    const l = layoutGraph(graph([], []), 'app')
    expect(l.nodes).toHaveLength(1)
    expect(l.edges).toHaveLength(0)
  })

  it('размеры холста покрывают все вершины', () => {
    const l = layoutGraph(graph([dep, rs], [edge(dep, rs)]), 'app')
    expect(l.width).toBe(Math.max(...l.nodes.map(n => n.x + n.w)))
    expect(l.height).toBe(Math.max(...l.nodes.map(n => n.y + n.h)))
  })
})

describe('collapseLeaves', () => {
  const many = (n: number, health = 'Healthy') => Array.from({ length: n }, (_, i) => pod(i, health))

  it(`схлопывает больше ${COLLAPSE_AT} подов с одними родителями в одну вершину`, () => {
    const pods = many(COLLAPSE_AT + 2)
    pods[3] = pod(3, 'Degraded')
    const g = graph([dep, rs, ...pods], [edge(dep, rs), ...pods.map(p => edge(rs, p))])
    const c = collapseLeaves(g, new Set())

    const group = c.nodes.find(n => n.id.startsWith(GROUP_PREFIX))!
    expect(group.count).toBe(COLLAPSE_AT + 2)
    expect(group.members).toHaveLength(COLLAPSE_AT + 2)
    expect(group.health).toBe('Degraded') // худший health группы виден на свёрнутой вершине
    expect(c.nodes.filter(n => n.kind === 'Pod')).toHaveLength(1)
    expect(c.edges.filter(e => e.to === group.id)).toHaveLength(1)
  })

  it('не трогает небольшие группы и развёрнутые группы', () => {
    const few = many(COLLAPSE_AT)
    const g1 = graph([rs, ...few], few.map(p => edge(rs, p)))
    expect(collapseLeaves(g1, new Set()).nodes).toHaveLength(1 + COLLAPSE_AT)

    const lots = many(COLLAPSE_AT + 3)
    const g2 = graph([rs, ...lots], lots.map(p => edge(rs, p)))
    const collapsed = collapseLeaves(g2, new Set())
    const gid = collapsed.nodes.find(n => n.id.startsWith(GROUP_PREFIX))!.id
    expect(collapseLeaves(g2, new Set([gid])).nodes).toHaveLength(1 + COLLAPSE_AT + 3)
  })

  it('сохраняет связи Service → Pod на группе без дублей рёбер', () => {
    const svc = node('Service', 'web')
    const pods = many(COLLAPSE_AT + 1)
    const g = graph(
      [rs, svc, ...pods],
      [...pods.map(p => edge(rs, p)), ...pods.map(p => edge(svc, p, 'selects'))],
    )
    const c = collapseLeaves(g, new Set())
    const gid = c.nodes.find(n => n.id.startsWith(GROUP_PREFIX))!.id
    const incoming = c.edges.filter(e => e.to === gid)
    expect(incoming.map(e => e.type).sort()).toEqual(['owns', 'selects'])
  })

  it('не схлопывает поды без родителей и не-поды', () => {
    const orphans = many(COLLAPSE_AT + 2)
    expect(collapseLeaves(graph(orphans, []), new Set()).nodes).toHaveLength(COLLAPSE_AT + 2)
    const cms = Array.from({ length: COLLAPSE_AT + 2 }, (_, i) => node('ConfigMap', `c${i}`))
    expect(collapseLeaves(graph([rs, ...cms], cms.map(c => edge(rs, c))), new Set()).nodes).toHaveLength(COLLAPSE_AT + 3)
  })
})

describe('filterGraph', () => {
  it('оставляет подходящие вершины вместе со всеми предками', () => {
    const healthy = pod(1)
    const bad = pod(2, 'Degraded')
    const other = node('ConfigMap', 'cfg')
    const g = graph([dep, rs, healthy, bad, other], [edge(dep, rs), edge(rs, healthy), edge(rs, bad)])
    const f = filterGraph(g, n => n.health === 'Degraded')

    expect(f.nodes.map(n => n.id).sort()).toEqual([dep.id, rs.id, bad.id].sort())
    expect(f.edges).toHaveLength(2)
  })

  it('пустой результат для фильтра без совпадений', () => {
    const f = filterGraph(graph([dep], []), () => false)
    expect(f.nodes).toHaveLength(0)
    expect(f.edges).toHaveLength(0)
  })
})

describe('worstHealth', () => {
  it('выбирает более тяжёлый статус', () => {
    expect(worstHealth('', 'Healthy')).toBe('Healthy')
    expect(worstHealth('Healthy', 'Progressing')).toBe('Progressing')
    expect(worstHealth('Progressing', 'Degraded')).toBe('Degraded')
    expect(worstHealth('Degraded', 'Healthy')).toBe('Degraded')
    expect(worstHealth('Healthy', 'Whatever')).toBe('Healthy')
  })
})

describe('collapseKinds', () => {
  const cms = (n: number, p: (i: number) => Partial<ArgoGraphNode> = () => ({})) =>
    Array.from({ length: n }, (_, i) => node('ConfigMap', `cm-${i}`, p(i)))

  it(`схлопывает больше ${KIND_COLLAPSE_AT} одиночных ресурсов одного Kind в одну вершину`, () => {
    const items = cms(KIND_COLLAPSE_AT + 1)
    const r = collapseKinds(items, [], new Set())
    expect(r.nodes).toHaveLength(1)
    const g = r.nodes[0] as ArgoGraphNode & { count?: number; members?: string[] }
    expect(g.id).toBe(KIND_GROUP_PREFIX + 'ConfigMap')
    expect(g.kind).toBe('ConfigMap')
    expect(g.name).toBe(`${KIND_COLLAPSE_AT + 1} × ConfigMap`)
    expect(g.count).toBe(KIND_COLLAPSE_AT + 1)
    expect(g.members).toHaveLength(KIND_COLLAPSE_AT + 1)
  })

  it('не трогает небольшие группы и развёрнутые', () => {
    expect(collapseKinds(cms(KIND_COLLAPSE_AT), [], new Set()).nodes).toHaveLength(KIND_COLLAPSE_AT)
    const items = cms(KIND_COLLAPSE_AT + 3)
    const r = collapseKinds(items, [], new Set([KIND_GROUP_PREFIX + 'ConfigMap']))
    expect(r.nodes).toHaveLength(KIND_COLLAPSE_AT + 3)
  })

  it('оставляет на виду проблемные ресурсы и считает порог без них', () => {
    const items = cms(KIND_COLLAPSE_AT + 2, i => (i === 0 ? { status: 'OutOfSync' } : i === 1 ? { health: 'Degraded' } : {}))
    // Здоровых ровно KIND_COLLAPSE_AT: группа не создаётся, все видны
    expect(collapseKinds(items, [], new Set()).nodes).toHaveLength(KIND_COLLAPSE_AT + 2)

    const more = [...items, node('ConfigMap', 'extra')]
    const r = collapseKinds(more, [], new Set())
    // 2 проблемных остаются отдельно + одна группа здоровых
    expect(r.nodes).toHaveLength(3)
    expect(r.nodes.filter(n => needsAttention(n))).toHaveLength(2)
  })

  it('не прячет ресурсы, у которых есть связи', () => {
    const items = cms(KIND_COLLAPSE_AT + 1)
    const owner = node('Deployment', 'd', { group: 'apps' })
    const r = collapseKinds([owner, ...items], [edge(owner, items[0])], new Set())
    expect(r.nodes.some(n => n.id === items[0].id)).toBe(true)
    // остальных (без связей) на одного меньше порога: группы нет
    expect(r.nodes).toHaveLength(items.length + 1)
  })

  it('layoutGraph подвешивает группу к корню одним ребром', () => {
    const items = cms(10)
    const l = layoutGraph(graph(items, []), 'app')
    expect(l.nodes).toHaveLength(2) // корень + группа
    expect(l.edges.filter(e => e.from === ROOT_ID)).toHaveLength(1)
    expect(l.nodes.find(n => n.id.startsWith(GROUP_PREFIX))?.count).toBe(10)
  })

  it('layoutGraph разворачивает группу по id', () => {
    const items = cms(10)
    const l = layoutGraph(graph(items, []), 'app', new Set([KIND_GROUP_PREFIX + 'ConfigMap']))
    expect(l.nodes).toHaveLength(11)
  })
})
