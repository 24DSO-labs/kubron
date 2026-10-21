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
}

export interface ArgoHistoryEntry {
  id: number
  revision: string
  deployedAt: string
  /** repoURL@targetRevision (path) */
  source: string
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

  message: string
  resources: ArgoResourceRef[]
  history: ArgoHistoryEntry[]

  labels: Record<string, string>
  annotations: Record<string, string>
  created: string
}
