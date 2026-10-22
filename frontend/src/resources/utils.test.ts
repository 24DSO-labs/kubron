import { describe, expect, it } from 'vitest'
import {
  containersOf,
  expandAccessModes,
  formatMap,
  itemName,
  itemNs,
  matchesSearch,
  podOwner,
  podStatusSummary,
  type PodOwnerKind,
} from './utils'

describe('itemName / itemNs', () => {
  it('понимает плоский и k8s-подобный формат', () => {
    expect(itemName({ name: 'a' })).toBe('a')
    expect(itemName({ metadata: { name: 'b' } })).toBe('b')
    expect(itemNs({ namespace: 'x' })).toBe('x')
    expect(itemNs({ metadata: { namespace: 'y' } })).toBe('y')
  })

  it('плоское поле приоритетнее metadata', () => {
    expect(itemName({ name: 'flat', metadata: { name: 'meta' } })).toBe('flat')
  })

  it('не падает на пустых значениях', () => {
    expect(itemName(null)).toBe('')
    expect(itemName(undefined)).toBe('')
    expect(itemNs({})).toBe('')
  })
})

describe('formatMap', () => {
  it('склеивает пары через запятую', () => {
    expect(formatMap({ app: 'web', tier: 'fe' })).toBe('app=web, tier=fe')
  })

  it('пусто для null/undefined/пустого объекта', () => {
    expect(formatMap(null)).toBe('')
    expect(formatMap(undefined)).toBe('')
    expect(formatMap({})).toBe('')
  })
})

describe('containersOf', () => {
  it('отбрасывает init-контейнеры', () => {
    const pod = { containers: [{ name: 'app' }, { name: 'init', isInit: true }, { name: 'sidecar', isInit: false }] }
    expect(containersOf(pod).map(c => c.name)).toEqual(['app', 'sidecar'])
  })

  it('пустой список, если контейнеров нет', () => {
    expect(containersOf({})).toEqual([])
    expect(containersOf(null)).toEqual([])
  })
})

describe('matchesSearch', () => {
  const item = { name: 'Nginx-Ingress', namespace: 'Kube-System' }

  it('пустой поиск (в т.ч. из пробелов) пропускает всё', () => {
    expect(matchesSearch(item, '')).toBe(true)
    expect(matchesSearch(item, '   ')).toBe(true)
  })

  it('ищет по имени и namespace без учёта регистра', () => {
    expect(matchesSearch(item, 'NGINX')).toBe(true)
    expect(matchesSearch(item, 'kube-sys')).toBe(true)
    expect(matchesSearch(item, ' ingress ')).toBe(true)
  })

  it('не находит чужое', () => {
    expect(matchesSearch(item, 'redis')).toBe(false)
  })
})

describe('podStatusSummary', () => {
  it('сортирует фазы по порядку Running → Pending → Succeeded → Failed → Unknown', () => {
    expect(podStatusSummary({ podStatuses: { Failed: 1, Running: 2, Pending: 3 } }))
      .toBe('Running: 2, Pending: 3, Failed: 1')
  })

  it('пропускает нулевые счётчики', () => {
    expect(podStatusSummary({ podStatuses: { Running: 1, Pending: 0 } })).toBe('Running: 1')
  })

  it('неизвестные фазы уходят в конец', () => {
    expect(podStatusSummary({ podStatuses: { Weird: 1, Unknown: 1, Running: 1 } }))
      .toBe('Running: 1, Unknown: 1, Weird: 1')
  })

  it('пустая строка без данных', () => {
    expect(podStatusSummary({})).toBe('')
    expect(podStatusSummary({ podStatuses: { Running: 0 } })).toBe('')
    expect(podStatusSummary(null)).toBe('')
  })
})

describe('podOwner', () => {
  it('deployment ищет его ReplicaSet по pod-template-hash', () => {
    const o = podOwner('deployment', 'web')
    expect(o.kind).toBe('ReplicaSet')
    expect(o.regex).toBe('web-[a-z0-9]{8,10}')
    const re = new RegExp(`^(?:${o.regex})$`)
    expect(re.test('web-5d8c7f9b6c')).toBe(true)
    expect(re.test('web-abc')).toBe(false)
    expect(re.test('web-api-5d8c7f9b6c')).toBe(false)
  })

  it.each<[PodOwnerKind, string]>([
    ['replicaset', 'ReplicaSet'],
    ['daemonset', 'DaemonSet'],
    ['statefulset', 'StatefulSet'],
    ['job', 'Job'],
  ])('%s -> kind %s, имя как есть', (kind, expected) => {
    expect(podOwner(kind, 'x')).toEqual({ kind: expected, regex: 'x' })
  })

  it('экранирует точки в имени, чтобы они не были «любым символом» в PromQL', () => {
    expect(podOwner('statefulset', 'db.v1').regex).toBe('db[.]v1')
    expect(podOwner('deployment', 'api.v2').regex).toBe('api[.]v2-[a-z0-9]{8,10}')
  })
})

describe('expandAccessModes', () => {
  it('раскрывает сокращения', () => {
    expect(expandAccessModes('RWO,RWX')).toBe('ReadWriteOnce, ReadWriteMany')
    expect(expandAccessModes('ROX')).toBe('ReadOnlyMany')
    expect(expandAccessModes('RWOP')).toBe('ReadWriteOncePod')
  })

  it('незнакомые значения оставляет как есть', () => {
    expect(expandAccessModes('RWO,FOO')).toBe('ReadWriteOnce, FOO')
  })

  it('пусто для undefined и пустой строки', () => {
    expect(expandAccessModes(undefined)).toBe('')
    expect(expandAccessModes('')).toBe('')
  })
})
