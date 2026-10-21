/** Контракты Argo CD: зеркало Go-структур internal/k8s/argocd_types.go */

export type ArgoSyncStatus = 'Synced' | 'OutOfSync' | 'Unknown'
export type ArgoHealthStatus =
  | 'Healthy' | 'Degraded' | 'Progressing' | 'Suspended' | 'Missing' | 'Unknown'

export interface ArgoResourceRef {
  group: string
  version: string
  kind: string
  namespace: string
  name: string
  status: ArgoSyncStatus | string
  health: ArgoHealthStatus | ''
  message: string
  hook: boolean
  requiresPruning: boolean
}

export interface ArgoHistoryEntry {
  id: number
  revision: string
  deployedAt: string
  /** repoURL@targetRevision (path) */
  source: string
}

/** Источник Application (spec.source или элемент spec.sources[]); repoURL без user:token@ */
export interface ArgoSourceInfo {
  repoURL: string
  path: string
  chart: string
  targetRevision: string
  /** имя ref-источника (multi-source) */
  ref: string
}

/** Condition Application: наличие в списке = активно (поля status у Argo нет, бэкенд ставит "True") */
export interface ArgoCondition {
  type: string
  status: string
  lastTransitionTime: string
  reason: string
  message: string
}

export interface ArgoApplicationInfo {
  name: string
  namespace: string
  project: string
  age: string

  syncStatus: ArgoSyncStatus | string
  healthStatus: ArgoHealthStatus | string
  revision: string

  repoURL: string
  path: string
  chart: string
  targetRevision: string

  destServer: string
  destName: string
  destNamespace: string

  autoSync: boolean
  selfHeal: boolean
  prune: boolean
  /** идёт операция (sync/rollback) */
  operationOn: boolean
  /** status.operationState.finishedAt (RFC3339); '' — синков не было */
  lastSyncAt: string
  /** Succeeded | Failed | Error | Running | '' */
  lastSyncPhase: string

  message: string
  /** все источники; repoURL/path/chart/targetRevision выше — первый из них */
  sources: ArgoSourceInfo[]
  conditions: ArgoCondition[]
  resources: ArgoResourceRef[]
  /** новые записи первыми */
  history: ArgoHistoryEntry[]

  labels: Record<string, string>
  annotations: Record<string, string>
  created: string
}
/** Параметры и результаты действий (зеркало Go: internal/k8s/argocd_actions.go, argocd_ui.go) */
export interface ArgoAppRef { namespace: string; name: string }
export interface ArgoSyncOptions { revision: string; prune: boolean; force: boolean; dryRun: boolean }
export interface ArgoActionResult { namespace: string; name: string; error: string }
export interface ArgoUIInfo { url: string; namespace: string; service: string; localPort: number; reused: boolean }