import type { ResourceDefinition } from './types'
import { role } from './definitions/role'
import { roleBinding } from './definitions/rolebinding'
import { clusterRole } from './definitions/clusterrole'
import { clusterRoleBinding } from './definitions/clusterrolebinding'
import { serviceAccount } from './definitions/serviceaccount'
import { configMap } from './definitions/configmap'
import { namespaceResource } from './definitions/namespace'
import { replicaSet } from './definitions/replicaset'
import { daemonSet } from './definitions/daemonset'
import { deployment } from './definitions/deployment'
import { pod } from './definitions/pod'
import { service } from './definitions/service'
import { secret } from './definitions/secret'
import { job } from './definitions/job'
import { statefulSet } from './definitions/statefulset'
import { cronJob } from './definitions/cronjob'
import { ingress } from './definitions/ingress'
import { ingressClass } from './definitions/ingressclass'
import { networkPolicy } from './definitions/networkpolicy'
import { persistentVolume } from './definitions/pv'
import { persistentVolumeClaim } from './definitions/pvc'
import { storageClass } from './definitions/storageclass'
import { resourceQuota } from './definitions/resourcequota'
import { limitRange } from './definitions/limitrange'
import { horizontalPodAutoscaler } from './definitions/hpa'
import { podDisruptionBudget } from './definitions/pdb'
import { priorityClass } from './definitions/priorityclass'
import { runtimeClass } from './definitions/runtimeclass'
import { lease } from './definitions/lease'
import { mutatingWebhookConfiguration, validatingWebhookConfiguration } from './definitions/webhook'
import { endpoint } from './definitions/endpoint'
import { event } from './definitions/event'
import { node } from './definitions/node'
import { portForward } from './definitions/portforward'
import { helmRelease } from './definitions/helmrelease'
import { helmChart } from './definitions/helmchart'
import { endpointSlice } from './definitions/endpointslice'
import { argoApplication } from './definitions/argocd/application'

/** Добавляя новый ресурс, достаточно дописать его сюда. */
export const resources: ResourceDefinition[] = [
  // Порядок задаёт порядок пунктов внутри групп бокового меню (см. navigation.ts)
  // workloads
  pod,
  deployment,
  daemonSet,
  statefulSet,
  replicaSet,
  job,
  cronJob,
  // config
  configMap,
  secret,
  resourceQuota,
  limitRange,
  horizontalPodAutoscaler,
  podDisruptionBudget,
  priorityClass,
  runtimeClass,
  lease,
  mutatingWebhookConfiguration,
  validatingWebhookConfiguration,
  // network
  service,
  endpoint,
  endpointSlice,
  ingress,
  ingressClass,
  networkPolicy,
  portForward,
  // storage
  persistentVolume,
  persistentVolumeClaim,
  storageClass,
  // helm
  helmRelease,
  helmChart,
  // argo cd
  argoApplication,
  // access control
  serviceAccount,
  role,
  roleBinding,
  clusterRole,
  clusterRoleBinding,
  // верхний уровень
  node,
  event,
  namespaceResource,
]

const byView = new Map(resources.map(r => [r.view, r]))
const byId = new Map(resources.map(r => [r.id, r]))

export const getResourceByView = (view: string) => byView.get(view)
export const getResourceById = (id: string) => byId.get(id.toLowerCase())
