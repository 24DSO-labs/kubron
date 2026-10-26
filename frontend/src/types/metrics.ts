// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

export interface MetricsSnapshot {
  timestamp: string
  nodesCpu?: unknown
  nodesMem?: unknown
  podsCpu?: unknown
  podsMem?: unknown
}

export interface ClusterMetricsSnapshot {
  cpuPercent: number
  cpuUsage: number
  cpuCapacity: number
  memoryPercent: number
  memoryUsage: number
  memoryCapacity: number
}

export interface NodeMetricSnapshot {
  name: string
  cpuPercent: number
  cpuUsage: number
  cpuCapacity: number
  memoryPercent: number
  memoryUsage: number
  memoryCapacity: number
  diskPercent: number
  diskUsage: number
  diskCapacity: number
}

export interface PodMetricSnapshot {
  name: string
  namespace: string
  cpuPercent: number
  cpuUsage: number
  memoryPercent: number
  memoryUsage: number
}

export interface MetricsStatus {
  status: 'not_installed' | 'installing' | 'installed' | 'degraded'
  components: Record<string, ComponentStatus>
  message?: string
  prometheusUrl?: string
}

export interface ComponentStatus {
  ready: boolean
  replicas: number
  availableReplicas: number
}

export interface PrometheusQueryResult {
  status: string
  data: PrometheusData
}

export interface PrometheusData {
  resultType: string
  result: PrometheusResult[]
}

export interface PrometheusResult {
  metric: Record<string, string>
  values: Array<[number, string]>
}

export type TimeSeriesDataPoint = [number, number]
