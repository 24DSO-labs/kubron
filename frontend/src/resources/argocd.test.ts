import { describe, expect, it } from 'vitest'
import type { ArgoApplicationInfo } from '../types/argocd'
import {
  argoCluster,
  argoDestination,
  argoPathOrChart,
  problemRank,
  shortRepo,
  shortRevision,
  summarizeArgo,
} from './argocd'

/** Минимальная заглушка Application: тесты задают только значимые поля */
const app = (p: Partial<ArgoApplicationInfo>): ArgoApplicationInfo => ({
  name: 'app',
  project: 'default',
  syncStatus: 'Synced',
  healthStatus: 'Healthy',
  lastSyncAt: '',
  operationOn: false,
  ...p,
} as ArgoApplicationInfo)

describe('shortRepo', () => {
  it('убирает схему и .git', () => {
    expect(shortRepo('https://github.com/acme/app.git')).toBe('github.com/acme/app')
    expect(shortRepo('ssh://git@host/acme/app')).toBe('git@host/acme/app')
    expect(shortRepo('git+https://host/x.git')).toBe('host/x')
  })

  it('пусто для пустого значения', () => {
    expect(shortRepo(undefined)).toBe('')
    expect(shortRepo('')).toBe('')
  })
})

describe('shortRevision', () => {
  it('sha сокращает до 7 символов', () => {
    expect(shortRevision('0123456789abcdef0123456789abcdef01234567')).toBe('0123456')
  })

  it('теги и ветки не трогает', () => {
    expect(shortRevision('v1.2.3')).toBe('v1.2.3')
    expect(shortRevision('main')).toBe('main')
    expect(shortRevision(undefined)).toBe('')
  })
})

describe('argoPathOrChart', () => {
  it('chart приоритетнее path', () => {
    expect(argoPathOrChart({ chart: 'nginx', path: 'x' })).toBe('nginx (chart)')
    expect(argoPathOrChart({ chart: '', path: 'deploy/prod' })).toBe('deploy/prod')
    expect(argoPathOrChart({ chart: '', path: '' })).toBe('')
  })
})

describe('argoDestination / argoCluster', () => {
  const inCluster = 'https://kubernetes.default.svc'

  it('имя кластера приоритетнее server', () => {
    expect(argoDestination({ destName: 'prod', destServer: 'https://x', destNamespace: 'web' })).toBe('prod / web')
    expect(argoCluster({ destName: 'prod', destServer: 'https://x' })).toBe('prod')
  })

  it('in-cluster сокращается', () => {
    expect(argoDestination({ destName: '', destServer: inCluster, destNamespace: 'web' })).toBe('in-cluster / web')
    expect(argoCluster({ destName: '', destServer: inCluster })).toBe('in-cluster')
  })

  it('без namespace — только кластер; без всего — unknown для группировки', () => {
    expect(argoDestination({ destName: 'prod', destServer: '', destNamespace: '' })).toBe('prod')
    expect(argoCluster({ destName: '', destServer: '' })).toBe('unknown')
  })
})

describe('problemRank', () => {
  const base: Parameters<typeof problemRank>[0] = { healthStatus: 'Healthy', syncStatus: 'Synced', lastSyncPhase: '', conditions: [] }

  it('здоровое приложение — 0', () => {
    expect(problemRank({ ...base })).toBe(0)
  })

  it('Degraded/Missing весят больше OutOfSync', () => {
    expect(problemRank({ ...base, healthStatus: 'Degraded' })).toBeGreaterThan(
      problemRank({ ...base, syncStatus: 'OutOfSync' }),
    )
    expect(problemRank({ ...base, healthStatus: 'Missing' })).toBe(4)
  })

  it('факторы суммируются', () => {
    expect(problemRank({
      healthStatus: 'Degraded',
      syncStatus: 'OutOfSync',
      lastSyncPhase: 'Failed',
      conditions: [{ type: 'ComparisonError' }],
    } as any)).toBe(4 + 2 + 3 + 2)
  })

  it('Progressing/Unknown — слабый сигнал', () => {
    expect(problemRank({ ...base, healthStatus: 'Progressing' })).toBe(1)
    expect(problemRank({ ...base, syncStatus: 'Unknown' })).toBe(1)
  })
})

describe('summarizeArgo', () => {
  const apps = [
    app({ name: 'ok', project: 'p1', destServer: 'https://kubernetes.default.svc', lastSyncAt: '2026-01-01T10:00:00Z' }),
    app({ name: 'drift', project: 'p1', syncStatus: 'OutOfSync', destName: 'prod', lastSyncAt: '2026-01-03T10:00:00Z' }),
    app({ name: 'broken', project: 'p2', healthStatus: 'Degraded', syncStatus: 'OutOfSync', destName: 'prod', operationOn: true }),
    app({ name: 'wip', project: '', healthStatus: 'Progressing', destName: 'prod', lastSyncAt: '2026-01-02T10:00:00Z' }),
  ]
  const s = summarizeArgo(apps)

  it('считает статусы', () => {
    expect(s.total).toBe(4)
    expect(s.synced).toBe(2)
    expect(s.outOfSync).toBe(2)
    expect(s.healthy).toBe(2)
    expect(s.degraded).toBe(1)
    expect(s.progressing).toBe(1)
    expect(s.syncing).toBe(1)
  })

  it('problems: только с problemRank > 0, самые проблемные первыми', () => {
    expect(s.problems.map(a => a.name)).toEqual(['broken', 'drift', 'wip'])
  })

  it('группировка по проекту: пустой проект -> default, проблемные выше', () => {
    // при равных problems — больше приложений выше, затем по имени
    expect(s.byProject).toEqual([
      { name: 'p1', total: 2, problems: 1 },
      { name: 'default', total: 1, problems: 1 },
      { name: 'p2', total: 1, problems: 1 },
    ])
  })

  it('группировка по кластеру', () => {
    expect(s.byCluster).toEqual([
      { name: 'prod', total: 3, problems: 3 },
      { name: 'in-cluster', total: 1, problems: 0 },
    ])
  })

  it('recentSyncs: новые первыми, без приложений без lastSyncAt, с лимитом', () => {
    expect(s.recentSyncs.map(a => a.name)).toEqual(['drift', 'wip', 'ok'])
    expect(summarizeArgo(apps, 1).recentSyncs.map(a => a.name)).toEqual(['drift'])
  })

  it('пустой список не ломается', () => {
    const e = summarizeArgo([])
    expect(e.total).toBe(0)
    expect(e.problems).toEqual([])
    expect(e.byProject).toEqual([])
  })
})
