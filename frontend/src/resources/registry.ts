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

/** Добавляя новый ресурс, достаточно дописать его сюда. */
export const resources: ResourceDefinition[] = [
  role,
  roleBinding,
  clusterRole,
  clusterRoleBinding,
  serviceAccount,
  configMap,
  namespaceResource,
  replicaSet,
  daemonSet,
  deployment,
  pod,
  service,
  secret,
  job,
  statefulSet,
]

const byView = new Map(resources.map(r => [r.view, r]))
const byId = new Map(resources.map(r => [r.id, r]))

export const getResourceByView = (view: string) => byView.get(view)
export const getResourceById = (id: string) => byId.get(id.toLowerCase())
