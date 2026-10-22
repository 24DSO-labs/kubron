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

/** Под, которому нужно внимание (internal/k8s/overview_insights.go) */
export interface ProblemPod {
  name: string
  namespace: string
  /** CrashLoopBackOff, ImagePullBackOff, Unschedulable, Pending, NotReady, Evicted ... */
  reason: string
  node: string
  restarts: number
  /** RFC3339: с какого момента под в этом состоянии */
  since: string
}

export interface ReasonCount {
  reason: string
  count: number
}

/** CPU в ядрах, память в байтах */
export interface ResourceRequests {
  cpuRequested: number
  cpuAllocatable: number
  memoryRequested: number
  memoryAllocatable: number
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

  /** контейнеры, перезапускавшиеся за 24 ч, и из них OOMKilled */
  restartsLast24h: number
  oomKilledLast24h: number
  /** до 10 самых проблемных подов и их общее число */
  problemPods: ProblemPod[]
  problemPodsTotal: number
  /** Warning-события за последний час и топ причин */
  warningsLastHour: number
  warningReasons: ReasonCount[]
  requests: ResourceRequests

  health: ClusterHealth
  recentEvents: OverviewEvent[]
  workloads: WorkloadSummary[]
}
