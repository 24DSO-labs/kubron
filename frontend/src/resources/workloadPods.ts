// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { callApp } from './wails'

interface Owner { kind: string; name: string }

/** Go отдаёт controlledBy как "Kind/name": "ReplicaSet/my-app-5d8c9f7b" */
function ownerOf(pod: any): Owner {
  const cb = String(pod?.controlledBy ?? '')
  const i = cb.indexOf('/')
  return i < 0 ? { kind: '', name: cb } : { kind: cb.slice(0, i), name: cb.slice(i + 1) }
}

function ownedBy(pods: any[], kind: string, names: Set<string>): any[] {
  return pods.filter(p => {
    const o = ownerOf(p)
    return o.kind === kind && names.has(o.name)
  })
}

/**
 * Поды ресурса по владельцу (ownerReferences), а не по имени пода:
 * имя пода обрезается для длинных имён ресурсов, владелец указан всегда.
 * kind — id ресурса в нижнем регистре ('deployment', 'daemonset', ...).
 */
export async function getWorkloadPods(kind: string, ns: string, name: string): Promise<any[]> {
  const pods = (((await callApp('GetPods', [ns])) as any[]) ?? []).filter(p => p.namespace === ns)

  switch (kind.toLowerCase()) {
    case 'pod':
      return pods.filter(p => p.name === name)
    case 'deployment': {
      // Deployment владеет ReplicaSet-ами: берём их точные имена, а не угадываем по префиксу
      const revisions = ((await callApp('GetDeploymentRevisions', ns, name)) as any[]) ?? []
      return ownedBy(pods, 'ReplicaSet', new Set(revisions.map(r => r.name)))
    }
    case 'replicaset':
      return ownedBy(pods, 'ReplicaSet', new Set([name]))
    case 'daemonset':
      return ownedBy(pods, 'DaemonSet', new Set([name]))
    case 'statefulset':
      return ownedBy(pods, 'StatefulSet', new Set([name]))
    case 'job':
      return ownedBy(pods, 'Job', new Set([name]))
    case 'cronjob': {
      const jobs = ((await callApp('GetCronJobJobs', ns, name)) as any[]) ?? []
      return ownedBy(pods, 'Job', new Set(jobs.map(j => j.name)))
    }
    default:
      return []
  }
}