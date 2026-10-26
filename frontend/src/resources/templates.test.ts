// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { manifestTemplate, hasTemplate, DEFAULT_NAMESPACE } from './templates'
import { runBatch } from './bulk'
import { toggledKey, prunedKeys } from './selection'

const def = (id: string, kind: string, namespaced: boolean) => ({ id, kind, namespaced })

describe('manifestTemplate', () => {
  it('namespaced: подставляет default и корректные apiVersion/kind', () => {
    const y = manifestTemplate(def('deployment', 'Deployment', true))!
    expect(y).toContain('apiVersion: apps/v1\nkind: Deployment\n')
    expect(y).toContain('  name: new-deployment\n  namespace: default\n')
    expect(DEFAULT_NAMESPACE).toBe('default')
  })

  it('кластерный ресурс: без namespace', () => {
    const y = manifestTemplate(def('clusterrole', 'ClusterRole', false))!
    expect(y).not.toContain('namespace:')
    expect(y).toContain('kind: ClusterRole')
  })

  it('metadata.name — валидное DNS-1123 имя', () => {
    const y = manifestTemplate(def('horizontalpodautoscaler', 'HorizontalPodAutoscaler', true))!
    expect(y).toContain('name: new-horizontalpodautoscaler')
  })

  it('короткие id hpa/pdb дают те же шаблоны', () => {
    expect(manifestTemplate(def('hpa', 'HorizontalPodAutoscaler', true))).toContain('apiVersion: autoscaling/v2')
    expect(manifestTemplate(def('pdb', 'PodDisruptionBudget', true))).toContain('apiVersion: policy/v1')
  })

  it('вложенность pod spec в Deployment корректна (containers под template.spec)', () => {
    const y = manifestTemplate(def('deployment', 'Deployment', true))!
    expect(y).toContain('    spec:\n      containers:\n        - name: main\n          image: nginx:stable\n')
  })

  it('Custom Resource: apiVersion из ключа cr:group/version/plural', () => {
    const d = def('cr:cert-manager.io/v1/certificates', 'Certificate', true)
    expect(hasTemplate(d)).toBe(true)
    const y = manifestTemplate(d)!
    expect(y).toContain('apiVersion: cert-manager.io/v1\nkind: Certificate\n')
    expect(y).toContain('spec: {}')
  })

  it('для типов без шаблона возвращает null', () => {
    expect(hasTemplate(def('node', 'Node', false))).toBe(false)
    expect(manifestTemplate(def('event', 'Event', true))).toBeNull()
  })

  it('манифест оканчивается переводом строки и не содержит табов', () => {
    for (const id of ['pod', 'service', 'secret', 'ingress', 'cronjob', 'role', 'storageclass']) {
      const y = manifestTemplate(def(id, id, true))!
      expect(y.endsWith('\n')).toBe(true)
      expect(y).not.toContain('\t')
    }
  })
})

describe('manifestTemplate: namespace', () => {
  it('Argo CD: Application создаётся в argocd, а не в default', () => {
    const y = manifestTemplate(def('application', 'Application', true))!
    expect(y).toContain('apiVersion: argoproj.io/v1alpha1\nkind: Application\n')
    expect(y).toContain('  namespace: argocd\n')
    expect(manifestTemplate(def('appproject', 'AppProject', true))).toContain('namespace: argocd')
  })

  it('явно переданный namespace важнее шаблонного', () => {
    expect(manifestTemplate(def('application', 'Application', true), 'tools')).toContain('namespace: tools')
    expect(manifestTemplate(def('pod', 'Pod', true), 'prod')).toContain('namespace: prod')
  })

  it('пустой namespace даёт default, а не пустую строку', () => {
    expect(manifestTemplate(def('pod', 'Pod', true), '')).toContain('  namespace: default\n')
  })

  it('каждый шаблон — kind и apiVersion на месте, имя и namespace не пустые', () => {
    const ids = ['pod','deployment','statefulset','daemonset','replicaset','job','cronjob','service','configmap','secret',
      'serviceaccount','ingress','networkpolicy','persistentvolumeclaim','persistentvolume','namespace','role','rolebinding',
      'clusterrole','clusterrolebinding','horizontalpodautoscaler','poddisruptionbudget','resourcequota','limitrange',
      'storageclass','ingressclass','priorityclass','runtimeclass','lease','application','appproject']
    for (const id of ids) {
      const y = manifestTemplate(def(id, id, true))!
      expect(y, id).toMatch(/^apiVersion: \S+\nkind: \S+\nmetadata:\n  name: new-[a-z0-9-]+\n/)
      expect(y, id).not.toMatch(/namespace:\s*$/m)
    }
  })
})

describe('selection helpers', () => {
  it('toggledKey не мутирует исходный набор', () => {
    const a = new Set(['x'])
    const b = toggledKey(a, 'y')
    expect([...a]).toEqual(['x'])
    expect([...b].sort()).toEqual(['x', 'y'])
    expect([...toggledKey(b, 'x')]).toEqual(['y'])
  })

  it('prunedKeys выкидывает пропавшие и не пересоздаёт набор без изменений', () => {
    const a = new Set(['default/a', 'default/b'])
    expect(prunedKeys(a, ['default/a', 'default/c'])).toEqual(new Set(['default/a']))
    expect(prunedKeys(a, ['default/a', 'default/b', 'x'])).toBe(a)
  })
})

describe('runBatch', () => {
  it('собирает успешные и упавшие, не прерываясь на ошибке', async () => {
    const r = await runBatch([1, 2, 3, 4], async n => {
      if (n % 2 === 0) throw new Error(`bad ${n}`)
    })
    expect(r.done.sort()).toEqual([1, 3])
    expect(r.failed.map(f => f.item).sort()).toEqual([2, 4])
    expect(r.failed[0].error).toMatch(/^bad/)
  })

  it('не превышает лимит параллелизма', async () => {
    let active = 0
    let peak = 0
    await runBatch(Array.from({ length: 10 }, (_, i) => i), async () => {
      active++
      peak = Math.max(peak, active)
      await new Promise(r => setTimeout(r, 5))
      active--
    }, 3)
    expect(peak).toBeLessThanOrEqual(3)
    expect(peak).toBeGreaterThan(1)
  })

  it('пустой список', async () => {
    expect(await runBatch([], async () => {})).toEqual({ done: [], failed: [] })
  })
})
