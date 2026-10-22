import { describe, expect, it } from 'vitest'
import { buildNodeAffinityYaml, buildPodAntiAffinityYaml } from './yaml'

describe('buildPodAntiAffinityYaml', () => {
  it('пусто без affinity', () => {
    expect(buildPodAntiAffinityYaml(null)).toEqual([])
    expect(buildPodAntiAffinityYaml({})).toEqual([])
  })

  it('preferred: weight, matchLabels и topologyKey с правильными отступами', () => {
    const out = buildPodAntiAffinityYaml({
      preferredDuringSchedulingIgnoredDuringExecution: [{
        weight: 100,
        podAffinityTerm: {
          labelSelector: { matchLabels: { app: 'web' } },
          topologyKey: 'kubernetes.io/hostname',
        },
      }],
    })
    expect(out).toEqual([
      'preferredDuringSchedulingIgnoredDuringExecution:',
      '  - weight: 100',
      '    podAffinityTerm:',
      '      labelSelector:',
      '        matchLabels:',
      '          app: web',
      '      topologyKey: kubernetes.io/hostname',
    ])
  })

  it('required без labelSelector не падает, topologyKey пустой', () => {
    const out = buildPodAntiAffinityYaml({
      requiredDuringSchedulingIgnoredDuringExecution: [{ podAffinityTerm: {} }],
    })
    expect(out).toEqual([
      'requiredDuringSchedulingIgnoredDuringExecution:',
      '  - labelSelector:',
      '    matchLabels:',
      '    topologyKey: ',
    ])
  })
})

describe('buildNodeAffinityYaml', () => {
  it('пусто без nodeAffinity', () => {
    expect(buildNodeAffinityYaml(undefined)).toEqual([])
    expect(buildNodeAffinityYaml({})).toEqual([])
  })

  it('required: matchExpressions со значениями', () => {
    const out = buildNodeAffinityYaml({
      nodeAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: {
          nodeSelectorTerms: [{
            matchExpressions: [{ key: 'disk', operator: 'In', values: ['ssd', 'nvme'] }],
          }],
        },
      },
    })
    expect(out).toEqual([
      'requiredDuringSchedulingIgnoredDuringExecution:',
      '  nodeSelectorTerms:',
      '    - matchExpressions:',
      '      - key: disk',
      '        operator: In',
      '        values:',
      '          - ssd',
      '          - nvme',
    ])
  })

  it('operator без values (Exists) не печатает values', () => {
    const out = buildNodeAffinityYaml({
      nodeAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: {
          nodeSelectorTerms: [{ matchExpressions: [{ key: 'gpu', operator: 'Exists' }] }],
        },
      },
    })
    expect(out).not.toContain('        values:')
    expect(out).toContain('        operator: Exists')
  })

  it('required и preferred разделяются пустой строкой', () => {
    const out = buildNodeAffinityYaml({
      nodeAffinity: {
        requiredDuringSchedulingIgnoredDuringExecution: {
          nodeSelectorTerms: [{ matchExpressions: [{ key: 'a', operator: 'Exists' }] }],
        },
        preferredDuringSchedulingIgnoredDuringExecution: [{
          weight: 5,
          preference: { matchExpressions: [{ key: 'b', operator: 'In', values: ['x'] }] },
        }],
      },
    })
    const i = out.indexOf('')
    expect(i).toBeGreaterThan(0)
    expect(out[i + 1]).toBe('preferredDuringSchedulingIgnoredDuringExecution:')
    expect(out).toContain('  - weight: 5')
  })
})
