// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

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
/** Приложение, созданное ApplicationSet */
export interface ArgoSetApp {
  name: string
  status: string
  health: string
  message: string
}

export interface ArgoApplicationSetInfo {
  name: string
  namespace: string
  age: string
  created: string

  /** типы генераторов: list, clusters, git, matrix ... */
  generators: string[]
  templateProject: string
  repoURL: string
  destNamespace: string
  syncPolicy: string
  preserveResourcesOnDeletion: boolean

  applications: ArgoSetApp[]
  appCount: number
  conditions: ArgoCondition[]
  /** нет активного ErrorOccurred */
  healthy: boolean

  labels: Record<string, string>
  annotations: Record<string, string>
}

export interface ArgoProjectDestination { server: string; name: string; namespace: string }
export interface ArgoGroupKind { group: string; kind: string }
export interface ArgoProjectRole {
  name: string
  description: string
  policies: string[]
  groups: string[]
}

export interface ArgoProjectInfo {
  name: string
  namespace: string
  description: string
  age: string
  created: string

  sourceRepos: string[]
  destinations: ArgoProjectDestination[]
  clusterResourceWhitelist: ArgoGroupKind[]
  clusterResourceBlacklist: ArgoGroupKind[]
  namespaceResourceWhitelist: ArgoGroupKind[]
  namespaceResourceBlacklist: ArgoGroupKind[]
  roles: ArgoProjectRole[]
  orphanedResourcesMonitoring: boolean

  labels: Record<string, string>
  annotations: Record<string, string>
}

/** Репозиторий Argo CD (из Secret). Санитизировано на Go: паролей, токенов и ключей здесь нет */
export interface ArgoRepositoryInfo {
  name: string
  namespace: string
  age: string
  repoName: string
  url: string
  /** git | helm | oci */
  type: string
  project: string
  /** anonymous | password | token | ssh | tls | github-app */
  authType: string
  insecure: boolean
  enableLfs: boolean
  enableOCI: boolean
  forceHttpBasicAuth: boolean
  /** задан proxy (адрес не отдаётся) */
  proxy: boolean
}

/** Кластер Argo CD (из Secret). Поле config с токенами и сертификатами наружу не отдаётся */
export interface ArgoClusterInfo {
  name: string
  namespace: string
  age: string
  clusterName: string
  server: string
  project: string
  /** none | token | tls | basic | aws | exec | unknown */
  authType: string
  insecure: boolean
  namespaces: string
  clusterResources: boolean
  inCluster: boolean
}

/** Вершина графа ресурсов приложения (internal/k8s/argocd_graph.go) */
export interface ArgoGraphNode {
  /** group/kind/namespace/name */
  id: string
  group: string
  kind: string
  namespace: string
  name: string
  status: string
  health: string
  message: string
  hook: boolean
  /** есть в status.resources приложения; false — найден по связям (ReplicaSet, Pod, чужой Service у Ingress) */
  managed: boolean
}

export type ArgoEdgeType = 'owns' | 'selects' | 'routes'

export interface ArgoGraphEdge {
  from: string
  to: string
  type: ArgoEdgeType
}

export interface ArgoGraph {
  nodes: ArgoGraphNode[]
  edges: ArgoGraphEdge[]
  /** достигнут предел числа вершин */
  truncated: boolean
  /** что не удалось прочитать (например, нет прав на pods) */
  warnings: string[]
}

/** Состояние опционального провайдера Argo CD API; сам токен фронтенду не отдаётся */
export interface ArgoAPIStatus {
  configured: boolean
}

export type ArgoDiffState = 'InSync' | 'Modified' | 'Missing' | 'Extra'

export interface ArgoResourceDiff {
  group: string
  kind: string
  namespace: string
  name: string
  hook: boolean
  state: ArgoDiffState
  /** unified diff «текущее → желаемое (git)»; значения Secret скрыты на бэкенде */
  diff: string
  /** diff обрезан или не построен из-за размера */
  truncated: boolean
}

export interface ArgoDiff {
  resources: ArgoResourceDiff[]
  total: number
  changed: number
  /** список ресурсов обрезан */
  truncated: boolean
}
