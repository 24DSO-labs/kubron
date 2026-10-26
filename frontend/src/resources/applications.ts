// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { podOwner, type PodOwnerKind } from './utils'

/** Workload приложения, как его отдаёт Go (k8s.ApplicationWorkload) */
export interface AppWorkload { kind: string; name: string; replicas?: number; ready?: number }

/** Kind workload-а -> PodOwnerKind (для PromQL kube_pod_owner) */
const OWNER_KINDS: Record<string, PodOwnerKind> = {
  Deployment: 'deployment',
  StatefulSet: 'statefulset',
  DaemonSet: 'daemonset',
}

/**
 * Поды приложения: те, чьи labels содержат все labels из selector приложения
 * (app.kubernetes.io/instance и app.kubernetes.io/name), в том же namespace.
 */
export function applicationPods(pods: any[], app: any): any[] {
  const selector = Object.entries((app?.selector ?? {}) as Record<string, string>)
  if (!selector.length) return []
  const ns = app?.namespace ?? ''
  return pods.filter(
    p => p.namespace === ns && selector.every(([k, v]) => p.labels?.[k] === v),
  )
}

/** Владельцы подов приложения для метрик: пары kube_pod_owner (owner_kind + regex имени) */
export function applicationOwners(app: any): { kind: string; regex: string }[] {
  const seen = new Set<string>()
  const owners: { kind: string; regex: string }[] = []
  for (const w of (app?.workloads ?? []) as AppWorkload[]) {
    const kind = OWNER_KINDS[w.kind]
    if (!kind) continue
    const o = podOwner(kind, w.name)
    const key = `${o.kind}/${o.regex}`
    if (seen.has(key)) continue
    seen.add(key)
    owners.push(o)
  }
  return owners
}

/** "ready/desired" по всем workload-ам приложения */
export function applicationReplicas(app: any): { ready: number; desired: number } {
  let ready = 0
  let desired = 0
  for (const w of (app?.workloads ?? []) as AppWorkload[]) {
    ready += w.ready ?? 0
    desired += w.replicas ?? 0
  }
  return { ready, desired }
}

/** Список URL для свойства панели: по строке на адрес, "None" если адресов нет (как в Lens) */
export function urlsText(urls?: string[] | null): string {
  return urls && urls.length ? urls.join('\n') : 'None'
}
