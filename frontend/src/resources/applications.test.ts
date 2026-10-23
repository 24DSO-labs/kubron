import { describe, expect, it } from 'vitest'
import { applicationOwners, applicationPods, applicationReplicas, urlsText } from './applications'

const app = {
  namespace: 'es',
  selector: { 'app.kubernetes.io/instance': 'ext', 'app.kubernetes.io/name': 'ext-webhook' },
  workloads: [
    { kind: 'Deployment', name: 'ext-webhook', replicas: 2, ready: 1 },
    { kind: 'StatefulSet', name: 'ext-db', replicas: 1, ready: 1 },
    { kind: 'CronJob', name: 'ignored', replicas: 0, ready: 0 },
  ],
}

describe('applicationPods', () => {
  const pod = (name: string, namespace: string, labels: Record<string, string>) => ({ name, namespace, labels })
  it('берёт поды namespace со всеми labels селектора', () => {
    const pods = [
      pod('a', 'es', { 'app.kubernetes.io/instance': 'ext', 'app.kubernetes.io/name': 'ext-webhook', x: '1' }),
      pod('b', 'es', { 'app.kubernetes.io/instance': 'ext', 'app.kubernetes.io/name': 'ext-core' }),
      pod('c', 'other', { 'app.kubernetes.io/instance': 'ext', 'app.kubernetes.io/name': 'ext-webhook' }),
      { name: 'd', namespace: 'es' },
    ]
    expect(applicationPods(pods, app).map(p => p.name)).toEqual(['a'])
  })
  it('без селектора подов нет (иначе подошёл бы весь namespace)', () => {
    expect(applicationPods([{ name: 'a', namespace: 'es', labels: {} }], { namespace: 'es', selector: {} })).toEqual([])
  })
})

describe('applicationOwners', () => {
  it('строит владельцев подов только для Deployment/StatefulSet/DaemonSet', () => {
    expect(applicationOwners(app)).toEqual([
      { kind: 'ReplicaSet', regex: 'ext-webhook-[a-z0-9]{8,10}' },
      { kind: 'StatefulSet', regex: 'ext-db' },
    ])
  })
  it('пустое приложение — пустой список', () => {
    expect(applicationOwners({})).toEqual([])
  })
})

describe('applicationReplicas / urlsText', () => {
  it('суммирует реплики workload-ов', () => {
    expect(applicationReplicas(app)).toEqual({ ready: 2, desired: 3 })
  })
  it('None без адресов', () => {
    expect(urlsText([])).toBe('None')
    expect(urlsText(undefined)).toBe('None')
    expect(urlsText(['a:80', 'b:81'])).toBe('a:80\nb:81')
  })
})
