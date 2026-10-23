/** Зеркало Go: internal/k8s/overview_hotspots.go */
export interface HotPod { namespace: string; name: string; used: number; percent: number }
export interface HotVolume { namespace: string; claim: string; used: number; capacity: number; percent: number }
export interface OverviewHotspots {
  topMemory: HotPod[]
  topCpu: HotPod[]
  volumes: HotVolume[]
  pvcPending: number
  pvcLost: number
}
export type MetricsState = 'loading' | 'ready' | 'not_installed' | 'installing' | 'degraded'
