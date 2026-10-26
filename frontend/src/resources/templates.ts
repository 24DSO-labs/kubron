// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import type { ResourceDefinition } from './types'

/**
 * Базовые манифесты для кнопки «+» в списках (создание объекта через YAML-редактор).
 * Ключ — ResourceDefinition.id. Ресурсов без записи здесь «+» не показывает:
 * их не создают руками (Node, Event, Endpoints), либо манифест слишком объёмный (CRD, webhooks).
 * Custom Resources (id вида "cr:group/version/plural") получают общий шаблон с пустым spec.
 */

/** Namespace по умолчанию для новых объектов (не зависит от выбранного в селекторе) */
export const DEFAULT_NAMESPACE = 'default'

interface Template {
  apiVersion: string
  kind: string
  /** Всё, что идёт после metadata: верхнеуровневые поля манифеста (spec, data, rules ...) */
  body?: string
  /** Namespace вместо DEFAULT_NAMESPACE: объекты Argo CD действуют только в его собственном namespace */
  namespace?: string
}

const T = (apiVersion: string, kind: string, body?: string, namespace?: string): Template => ({
  apiVersion,
  kind,
  body,
  namespace,
})

/** Отступ блока: шаблоны ниже записаны без лишних пробелов слева */
const lines = (s: string) => s.replace(/^\n/, '').replace(/\s+$/, '')

const podSpec = (indent: string) =>
  lines(`
containers:
  - name: main
    image: nginx:stable
    ports:
      - containerPort: 80
`)
    .split('\n')
    .map(l => indent + l)
    .join('\n')

const TEMPLATES: Record<string, Template> = {
  pod: T('v1', 'Pod', `spec:\n${podSpec('  ')}`),

  deployment: T('apps/v1', 'Deployment', lines(`
spec:
  replicas: 1
  selector:
    matchLabels:
      app: new-deployment
  template:
    metadata:
      labels:
        app: new-deployment
    spec:
${podSpec('      ')}
`)),

  statefulset: T('apps/v1', 'StatefulSet', lines(`
spec:
  serviceName: new-statefulset
  replicas: 1
  selector:
    matchLabels:
      app: new-statefulset
  template:
    metadata:
      labels:
        app: new-statefulset
    spec:
${podSpec('      ')}
`)),

  daemonset: T('apps/v1', 'DaemonSet', lines(`
spec:
  selector:
    matchLabels:
      app: new-daemonset
  template:
    metadata:
      labels:
        app: new-daemonset
    spec:
${podSpec('      ')}
`)),

  replicaset: T('apps/v1', 'ReplicaSet', lines(`
spec:
  replicas: 1
  selector:
    matchLabels:
      app: new-replicaset
  template:
    metadata:
      labels:
        app: new-replicaset
    spec:
${podSpec('      ')}
`)),

  job: T('batch/v1', 'Job', lines(`
spec:
  backoffLimit: 3
  template:
    spec:
      restartPolicy: Never
      containers:
        - name: main
          image: busybox:stable
          command: ["sh", "-c", "echo hello"]
`)),

  cronjob: T('batch/v1', 'CronJob', lines(`
spec:
  schedule: "*/5 * * * *"
  jobTemplate:
    spec:
      template:
        spec:
          restartPolicy: OnFailure
          containers:
            - name: main
              image: busybox:stable
              command: ["sh", "-c", "date"]
`)),

  service: T('v1', 'Service', lines(`
spec:
  type: ClusterIP
  selector:
    app: my-app
  ports:
    - name: http
      port: 80
      targetPort: 80
`)),

  configmap: T('v1', 'ConfigMap', lines(`
data:
  key: value
`)),

  secret: T('v1', 'Secret', lines(`
type: Opaque
stringData:
  key: value
`)),

  serviceaccount: T('v1', 'ServiceAccount'),

  ingress: T('networking.k8s.io/v1', 'Ingress', lines(`
spec:
  rules:
    - host: example.local
      http:
        paths:
          - path: /
            pathType: Prefix
            backend:
              service:
                name: my-service
                port:
                  number: 80
`)),

  networkpolicy: T('networking.k8s.io/v1', 'NetworkPolicy', lines(`
spec:
  podSelector: {}
  policyTypes:
    - Ingress
`)),

  persistentvolumeclaim: T('v1', 'PersistentVolumeClaim', lines(`
spec:
  accessModes:
    - ReadWriteOnce
  resources:
    requests:
      storage: 1Gi
`)),

  persistentvolume: T('v1', 'PersistentVolume', lines(`
spec:
  capacity:
    storage: 1Gi
  accessModes:
    - ReadWriteOnce
  persistentVolumeReclaimPolicy: Retain
  hostPath:
    path: /data/new-persistentvolume
`)),

  namespace: T('v1', 'Namespace'),

  role: T('rbac.authorization.k8s.io/v1', 'Role', lines(`
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
`)),

  rolebinding: T('rbac.authorization.k8s.io/v1', 'RoleBinding', lines(`
subjects:
  - kind: ServiceAccount
    name: default
    namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: Role
  name: my-role
`)),

  clusterrole: T('rbac.authorization.k8s.io/v1', 'ClusterRole', lines(`
rules:
  - apiGroups: [""]
    resources: ["pods"]
    verbs: ["get", "list", "watch"]
`)),

  clusterrolebinding: T('rbac.authorization.k8s.io/v1', 'ClusterRoleBinding', lines(`
subjects:
  - kind: ServiceAccount
    name: default
    namespace: default
roleRef:
  apiGroup: rbac.authorization.k8s.io
  kind: ClusterRole
  name: my-clusterrole
`)),

  horizontalpodautoscaler: T('autoscaling/v2', 'HorizontalPodAutoscaler', lines(`
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: Deployment
    name: my-deployment
  minReplicas: 1
  maxReplicas: 3
  metrics:
    - type: Resource
      resource:
        name: cpu
        target:
          type: Utilization
          averageUtilization: 80
`)),

  poddisruptionbudget: T('policy/v1', 'PodDisruptionBudget', lines(`
spec:
  minAvailable: 1
  selector:
    matchLabels:
      app: my-app
`)),

  resourcequota: T('v1', 'ResourceQuota', lines(`
spec:
  hard:
    pods: "10"
    requests.cpu: "4"
    requests.memory: 8Gi
`)),

  limitrange: T('v1', 'LimitRange', lines(`
spec:
  limits:
    - type: Container
      default:
        cpu: 500m
        memory: 256Mi
      defaultRequest:
        cpu: 100m
        memory: 128Mi
`)),

  storageclass: T('storage.k8s.io/v1', 'StorageClass', lines(`
provisioner: kubernetes.io/no-provisioner
reclaimPolicy: Delete
volumeBindingMode: WaitForFirstConsumer
`)),

  ingressclass: T('networking.k8s.io/v1', 'IngressClass', lines(`
spec:
  controller: k8s.io/ingress-nginx
`)),

  priorityclass: T('scheduling.k8s.io/v1', 'PriorityClass', lines(`
value: 1000
globalDefault: false
description: Describe when this priority class should be used
`)),

  runtimeclass: T('node.k8s.io/v1', 'RuntimeClass', lines(`
handler: runc
`)),

  lease: T('coordination.k8s.io/v1', 'Lease', lines(`
spec:
  holderIdentity: me
  leaseDurationSeconds: 30
`)),
}

// Argo CD: объекты живут в namespace самого Argo CD
const ARGO_NAMESPACE = 'argocd'

TEMPLATES.application = T('argoproj.io/v1alpha1', 'Application', lines(`
spec:
  project: default
  source:
    repoURL: https://github.com/argoproj/argocd-example-apps.git
    targetRevision: HEAD
    path: guestbook
  destination:
    server: https://kubernetes.default.svc
    namespace: default
`), ARGO_NAMESPACE)

TEMPLATES.appproject = T('argoproj.io/v1alpha1', 'AppProject', lines(`
spec:
  description: New project
  sourceRepos:
    - "*"
  destinations:
    - server: https://kubernetes.default.svc
      namespace: "*"
`), ARGO_NAMESPACE)

// Короткие id тех же типов (в определениях ресурсов встречаются оба варианта)
TEMPLATES.hpa = TEMPLATES.horizontalpodautoscaler
TEMPLATES.pdb = TEMPLATES.poddisruptionbudget

const CUSTOM_PREFIX = 'cr:'

/** "cr:group/version/plural" -> apiVersion "group/version" */
function customApiVersion(id: string): string | null {
  const [group, version] = id.slice(CUSTOM_PREFIX.length).split('/')
  return group && version ? `${group}/${version}` : null
}

type DefLike = Pick<ResourceDefinition, 'id' | 'kind' | 'namespaced'>

function templateFor(def: DefLike): Template | null {
  const known = TEMPLATES[def.id]
  if (known) return known
  if (def.id.startsWith(CUSTOM_PREFIX)) {
    const apiVersion = customApiVersion(def.id)
    if (apiVersion) return T(apiVersion, def.kind, 'spec: {}')
  }
  return null
}

/** Можно ли создавать ресурсы этого типа из шаблона */
export const hasTemplate = (def: DefLike): boolean => templateFor(def) !== null

/**
 * Базовый манифест нового объекта. null — для типа нет шаблона.
 * У кластерных ресурсов namespace не указывается.
 */
export function manifestTemplate(def: DefLike, namespace: string = DEFAULT_NAMESPACE): string | null {
  const t = templateFor(def)
  if (!t) return null

  // Имя должно быть валидным DNS-1123: kind в нижнем регистре, без спецсимволов
  const name = `new-${t.kind.toLowerCase().replace(/[^a-z0-9]+/g, '-')}`
  const out = [
    `apiVersion: ${t.apiVersion}`,
    `kind: ${t.kind}`,
    'metadata:',
    `  name: ${name}`,
  ]
  // Явно переданный namespace важнее шаблонного (его выбирает вызывающий код), шаблонный — важнее default
  if (def.namespaced) {
    const ns = namespace && namespace !== DEFAULT_NAMESPACE ? namespace : (t.namespace ?? DEFAULT_NAMESPACE)
    out.push(`  namespace: ${ns}`)
  }
  if (t.body) out.push(t.body)
  return out.join('\n') + '\n'
}
