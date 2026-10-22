/** Зеркало Go: internal/k8s/crd.go */
export interface CRDColumn { name: string; type: string; jsonPath: string; description: string; priority: number }

export interface CRDInfo {
  name: string
  group: string
  kind: string
  plural: string
  singular: string
  scope: 'Namespaced' | 'Cluster' | string
  version: string
  versions: string[]
  shortNames: string[]
  categories: string[]
  columns: CRDColumn[]
  conditions: { type: string; status: string; reason: string; message: string; lastTransitionTime: string }[]
  age: string
  created: string
  labels: Record<string, string>
  annotations: Record<string, string>
  controlledBy: string
  managedBy: string
}

export interface CustomResourceInfo {
  name: string
  namespace: string
  apiVersion: string
  kind: string
  age: string
  created: string
  /** значения printer-колонок по имени колонки */
  cells: Record<string, string>
  conditions: { type: string; status: string; reason: string; message: string; lastTransitionTime: string }[]
  finalizers: string[]
  labels: Record<string, string>
  annotations: Record<string, string>
  controlledBy: string
  managedBy: string
}