// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { describe, expect, it } from 'vitest'
import { mergePodMetrics, podCpuCol, podMemoryCol } from './podUsage'
const metric = (name: string, totalCpu: string, totalMem: string) =>
  ({ name, namespace: 'default', containers: [], totalCpu, totalMem })

describe('mergePodMetrics', () => {
  const pods = [
    { name: 'a', namespace: 'default' },
    { name: 'b', namespace: 'default' },
    { name: 'a', namespace: 'other' },
  ]
  const merged = mergePodMetrics(pods, [
    metric('a', '12.34', '125829120'), // 12.34m, 120 MiB
    metric('b', '1500', '1073741824'), // 1.5 ядра, 1024 MiB
  ])

  it('сопоставляет по namespace и имени', () => {
    expect(merged[0].cpuMillicores).toBe(12.34)
    expect(merged[1].cpuMillicores).toBe(1500)
    expect(merged[2].cpuMillicores).toBeNull() // тот же name в другом namespace
  })

  it('сохраняет сырые числа для сортировки и готовый текст для ячейки', () => {
    expect(merged[0]).toMatchObject({ cpu: '12.3m', memory: '120.0MiB', memoryBytes: 125829120 })
    expect(merged[1]).toMatchObject({ cpu: '1.50', memory: '1024.0MiB' })
  })

  it('без метрик: прочерк и null', () => {
    expect(merged[2]).toMatchObject({ cpu: '—', memory: '—', cpuMillicores: null, memoryBytes: null })
  })
})

describe('колонки CPU / Memory пода', () => {
  const col = (key: string) => (key === 'cpu' ? podCpuCol : podMemoryCol)
  const [a, b, none] = mergePodMetrics(
    [{ name: 'a', namespace: 'n' }, { name: 'b', namespace: 'n' }, { name: 'c', namespace: 'n' }],
    [
      { name: 'a', namespace: 'n', containers: [], totalCpu: '900', totalMem: '2097152' },
      { name: 'b', namespace: 'n', containers: [], totalCpu: '1500', totalMem: '1073741824' },
    ],
  )

  it('показывает текст как есть, без повторного форматирования', () => {
    expect(col('cpu').formatter!(b as any)).toBe('1.50') // раньше превращалось в «1.5m»
    expect(col('memory').formatter!(b as any)).toBe('1024.0MiB') // раньше — «0.0MiB»
    expect(col('cpu').formatter!(none as any)).toBe('—')
  })

  it('сортирует по числам, а не по тексту', () => {
    // как строки "1.50" < "900.0m", как числа 1500 > 900
    expect(col('cpu').sortValue!(a as any)).toBeLessThan(col('cpu').sortValue!(b as any) as number)
    expect(col('memory').sortValue!(a as any)).toBeLessThan(col('memory').sortValue!(b as any) as number)
    expect(col('cpu').sortValue!(none as any)).toBeNull()
  })
})
