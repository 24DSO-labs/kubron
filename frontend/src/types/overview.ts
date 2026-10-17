/** Данные главного экрана: форма совпадает с k8s.ClusterOverview из Go (internal/k8s/overview.go) */

export type HealthStatus = 'healthy' | 'unhealthy' | 'unknown'

export interface HealthComponent {
  name: string
  status: HealthStatus
}

export interface ClusterHealth {
  /** 0..100 */
  score: number
  status: 'Healthy' | 'Degraded' | 'Critical'
  components: HealthComponent[]
}

export interface OverviewEvent {
  type: 'Normal' | 'Warning' | string
  reason: string
  message: string
  /** Kind/name */
  object: string
  namespace: string
  /** RFC3339 */
  time: string
}

export type WorkloadKind = 'Deployment' | 'StatefulSet' | 'DaemonSet'

export interface WorkloadSummary {
  name: string
  namespace: string
  kind: WorkloadKind
  status: 'Running' | 'Degraded' | 'Unavailable' | 'Idle'
  ready: number
  desired: number
  upToDate: number
  age: string
}

export interface ClusterOverview {
  totalNodes: number
  readyNodes: number
  totalPods: number
  runningPods: number
  pendingPods: number
  failedPods: number
  totalDeployments: number
  readyDeployments: number
  totalNamespaces: number
  activeNamespaces: number
  totalEvents: number
  warningEvents: number
  health: ClusterHealth
  recentEvents: OverviewEvent[]
  workloads: WorkloadSummary[]
}
