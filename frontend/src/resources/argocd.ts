/** Чистые функции Argo CD: подписи, приоритет «проблемности» и сводка для дашборда (без Vue и Wails) */
import type { ArgoApplicationInfo } from '../types/argocd'

/** repoURL без схемы и .git: https://github.com/acme/app.git -> github.com/acme/app */
export function shortRepo(url?: string): string {
  if (!url) return ''
  return url.replace(/^[a-z+]+:\/\//i, '').replace(/\.git$/, '')
}

/** Путь в репозитории или имя Helm-чарта */
export function argoPathOrChart(a: Pick<ArgoApplicationInfo, 'path' | 'chart'>): string {
  return a.chart ? `${a.chart} (chart)` : a.path || ''
}

export function shortRevision(rev?: string): string {
  if (!rev) return ''
  // sha (40 hex) сокращаем до 7 символов, теги и ветки оставляем как есть
  return /^[0-9a-f]{40}$/i.test(rev) ? rev.slice(0, 7) : rev
}

/** Кластер назначения: имя, иначе server (in-cluster сокращаем), + namespace */
export function argoDestination(a: Pick<ArgoApplicationInfo, 'destName' | 'destServer' | 'destNamespace'>): string {
  const cluster = a.destName || (a.destServer === 'https://kubernetes.default.svc' ? 'in-cluster' : a.destServer)
  return [cluster, a.destNamespace].filter(Boolean).join(' / ')
}

/** Метка кластера назначения (без namespace), для группировки на дашборде */
export function argoCluster(a: Pick<ArgoApplicationInfo, 'destName' | 'destServer'>): string {
  return a.destName || (a.destServer === 'https://kubernetes.default.svc' ? 'in-cluster' : a.destServer) || 'unknown'
}

/** 0 — всё хорошо; чем больше, тем приоритетнее показать приложение (Degraded/Missing выше OutOfSync) */
export function problemRank(a: Pick<ArgoApplicationInfo, 'healthStatus' | 'syncStatus' | 'lastSyncPhase' | 'conditions'>): number {
  let r = 0
  if (a.healthStatus === 'Degraded' || a.healthStatus === 'Missing') r += 4
  else if (a.healthStatus === 'Progressing' || a.healthStatus === 'Unknown') r += 1
  if (a.syncStatus === 'OutOfSync') r += 2
  else if (a.syncStatus === 'Unknown') r += 1
  if (a.lastSyncPhase === 'Failed' || a.lastSyncPhase === 'Error') r += 3
  if (a.conditions?.length) r += 2
  return r
}

export interface ArgoSummary {
  total: number
  synced: number
  outOfSync: number
  healthy: number
  degraded: number
  progressing: number
  suspended: number
  missing: number
  unknown: number
  syncing: number
  /** приложения с problemRank > 0, самые проблемные первыми */
  problems: ArgoApplicationInfo[]
  byProject: { name: string; total: number; problems: number }[]
  byCluster: { name: string; total: number; problems: number }[]
  /** последние синки (по lastSyncAt, новые первыми) */
  recentSyncs: ArgoApplicationInfo[]
}

function group(apps: ArgoApplicationInfo[], key: (a: ArgoApplicationInfo) => string) {
  const m = new Map<string, { name: string; total: number; problems: number }>()
  for (const a of apps) {
    const k = key(a)
    const g = m.get(k) ?? { name: k, total: 0, problems: 0 }
    g.total++
    if (problemRank(a) > 0) g.problems++
    m.set(k, g)
  }
  return [...m.values()].sort((x, y) => y.problems - x.problems || y.total - x.total || x.name.localeCompare(y.name))
}

export function summarizeArgo(apps: ArgoApplicationInfo[], recentLimit = 8): ArgoSummary {
  const count = (pred: (a: ArgoApplicationInfo) => boolean) => apps.filter(pred).length
  const byTime = (a: ArgoApplicationInfo) => Date.parse(a.lastSyncAt) || 0
  return {
    total: apps.length,
    synced: count(a => a.syncStatus === 'Synced'),
    outOfSync: count(a => a.syncStatus === 'OutOfSync'),
    healthy: count(a => a.healthStatus === 'Healthy'),
    degraded: count(a => a.healthStatus === 'Degraded'),
    progressing: count(a => a.healthStatus === 'Progressing'),
    suspended: count(a => a.healthStatus === 'Suspended'),
    missing: count(a => a.healthStatus === 'Missing'),
    unknown: count(a => a.healthStatus === 'Unknown'),
    syncing: count(a => a.operationOn),
    problems: apps
      .filter(a => problemRank(a) > 0)
      .sort((x, y) => problemRank(y) - problemRank(x) || x.name.localeCompare(y.name)),
    byProject: group(apps, a => a.project || 'default'),
    byCluster: group(apps, argoCluster),
    recentSyncs: apps
      .filter(a => byTime(a) > 0)
      .sort((x, y) => byTime(y) - byTime(x))
      .slice(0, recentLimit),
  }
}
