export interface ClusterMeta {
  id: string
  displayName: string
  icon: string
  color: string
  kubeconfigFile: string
  defaultNamespace: string
  addedAt: string
  contextName: string
}

export interface ClusterStatus {
  id: string
  displayName: string
  icon: string
  color: string
  kubeconfigFile: string
  defaultNamespace: string
  addedAt: string
  contextName: string
  connected: boolean
  active: boolean
  error: string
}

export interface ContextInfo {
  name: string
  cluster: string
  user: string
  namespace: string
  isCurrent: boolean
}

export interface NamespaceInfo {
  name: string
  status: string
  age: string
}

export interface PodInfo {
  name: string
  namespace: string
  status: string
  ready: string
  restarts: number
  age: string
  node: string
  ip: string
}

export interface ContainerInfo {
  name: string
  image: string
  ready: boolean
  restartCount: number
  state: string
}

export interface PodDetailInfo {
  name: string
  namespace: string
  status: string
  node: string
  ip: string
  labels: Record<string, string>
  age: string
  containers: ContainerInfo[]
}

export interface DeploymentInfo {
  name: string
  namespace: string
  replicas: number
  readyReplicas: number
  updatedReplicas: number
  age: string
  strategy: string
  images: string[]
}

export interface StatefulSetInfo {
  name: string
  namespace: string
  replicas: number
  readyReplicas: number
  age: string
  images: string[]
  serviceName: string
}

export interface ReplicaSetInfo {
  name: string
  namespace: string
  desired: number
  ready: number
  age: string
  images: string[]
}

export interface DaemonSetInfo {
  name: string
  namespace: string
  desired: number
  current: number
  ready: number
  available: number
  age: string
  images: string[]
}

export interface JobInfo {
  name: string
  namespace: string
  completions: string
  succeeded: number
  failed: number
  active: number
  duration: string
  age: string
  status: string
}

export interface CronJobInfo {
  name: string
  namespace: string
  schedule: string
  suspend: boolean
  active: number
  lastSchedule: string
  age: string
}

export interface ServiceInfo {
  name: string
  namespace: string
  type: string
  clusterIP: string
  externalIP: string
  ports: string[]
  age: string
}

export interface IngressInfo {
  name: string
  namespace: string
  class: string
  hosts: string[]
  paths: string
  address: string
  age: string
  tls: boolean
}

export interface NodeInfo {
  name: string
  status: string
  roles: string
  version: string
  os: string
  arch: string
  cpuCapacity: string
  memoryCapacity: string
  internalIP: string
  age: string
}

export interface EventInfo {
  type: string
  reason: string
  message: string
  object: string
  count: number
  age: string
  source: string
  namespace: string
}

export interface ConfigMapInfo {
  name: string
  namespace: string
  dataCount: number
  age: string
  labels: any
  data: Record<string, string>
}

export interface SecretInfo {
  name: string
  namespace: string
  type: string
  dataCount: number
  age: string
  keys: string[]
}

export interface PVInfo {
  name: string
  capacity: string
  accessModes: string
  reclaimPolicy: string
  status: string
  claim: string
  storageClass: string
  age: string
}

export interface PVCInfo {
  name: string
  namespace: string
  status: string
  volume: string
  capacity: string
  accessModes: string
  storageClass: string
  age: string
}

export interface OverviewInfo {
  totalNodes: number
  readyNodes: number
  totalPods: number
  runningPods: number
  pendingPods: number
  failedPods: number
  totalDeployments: number
  readyDeployments: number
  totalServices: number
  totalNamespaces: number
  totalEvents: number
  warningEvents: number
}

export interface PodMetricsInfo {
  name: string
  namespace: string
  containers: ContainerMetricsInfo[]
  totalCpu: string
  totalMem: string
}

export interface ContainerMetricsInfo {
  name: string
  cpu: string
  memory: string
}

export interface NodeMetricsInfo {
  name: string
  cpuUsage: string
  memUsage: string
  cpuPct: number
  memPct: number
}

export interface RevisionInfo {
  revision: string
  name: string
  replicas: number
  ready: number
  age: string
  images: string
}

export interface PortForwardSession {
  id: string
  namespace: string
  pod: string
  localPort: number
  podPort: number
  status: string
}

export interface ServiceDetail {
  name: string
  namespace: string
  type: string
  clusterIP: string
  clusterIPs: string[]
  ipFamilies: string
  ipFamilyPolicy: string
  ports: ServicePortInfo[]
  selector: Record<string, string>
  labels: Record<string, string>
  annotations: Record<string, string>
  age: string
  endpoints: EndpointInfo[]
}

export interface ServicePortInfo {
  name: string
  port: number
  targetPort: string
  protocol: string
  nodePort: number
  forwarded: boolean
  localPort: number
}

export interface EndpointInfo {
  name: string
  addresses: string[]
}
export interface TerminalOutput {
  stdout: string
  stderr: string
  exitCode: number
}

export interface ResourceQuotaInfo {
  name: string
  namespace: string
  age: string
  cpuLimit: string
  cpuUsed: string
  memLimit: string
  memUsed: string
}

export interface LimitRangeInfo {
  name: string
  namespace: string
  age: string
  items: number
}

export interface HPAInfo {
  name: string
  namespace: string
  reference: string
  minReplicas: number
  maxReplicas: number
  currentReplicas: number
  desiredReplicas: number
  age: string
}

export interface PDBInfo {
  name: string
  namespace: string
  minAvailable: string
  maxUnavailable: string
  disruptionsAllowed: number
  currentHealthy: number
  desiredHealthy: number
  age: string
}

export interface PriorityClassInfo {
  name: string
  value: number
  age: string
}

export interface RuntimeClassInfo {
  name: string
  handler: string
  age: string
}

export interface LeaseInfo {
  name: string
  namespace: string
  holder: string
  age: string
}

export interface EndpointListInfo {
  name: string
  namespace: string
  addresses: number
  ports: number
  age: string
}

export interface NetworkPolicyInfo {
  name: string
  namespace: string
  podSelector: string
  age: string
}

export interface IngressClassInfo {
  name: string
  controller: string
  age: string
}

export interface PortForwardingInfo {
  id: string
  namespace: string
  pod: string
  localPort: number
  podPort: number
  status: string
}

export interface StorageClassInfo {
  name: string
  provisioner: string
  reclaimPolicy: string
  default: boolean
  age: string
}

export interface ServiceAccount {
  apiVersion?: string
  kind?: string
  metadata?: {
    name?: string
    namespace?: string
    creationTimestamp?: string
  }
  secrets?: Array<{
    name: string
  }>
  automountServiceAccountToken?: boolean
}

export interface Role {
  apiVersion?: string
  kind?: string
  metadata?: {
    name?: string
    namespace?: string
    creationTimestamp?: string
  }
  rules?: Array<{
    verbs: string[]
    apiGroups?: string[]
    resources?: string[]
    resourceNames?: string[]
  }>
}

export interface RoleBinding {
  apiVersion?: string
  kind?: string
  metadata?: {
    name?: string
    namespace?: string
    creationTimestamp?: string
  }
  roleRef?: {
    apiGroup: string
    kind: string
    name: string
  }
  subjects?: Array<{
    apiGroup?: string
    kind: string
    name: string
    namespace?: string
  }>
}

export interface ClusterRole {
  apiVersion?: string
  kind?: string
  metadata?: {
    name?: string
    creationTimestamp?: string
  }
  rules?: Array<{
    verbs: string[]
    apiGroups?: string[]
    resources?: string[]
    resourceNames?: string[]
  }>
}

export interface ClusterRoleBinding {
  apiVersion?: string
  kind?: string
  metadata?: {
    name?: string
    creationTimestamp?: string
  }
  roleRef?: {
    apiGroup: string
    kind: string
    name: string
  }
  subjects?: Array<{
    apiGroup?: string
    kind: string
    name: string
    namespace?: string
  }>
}
