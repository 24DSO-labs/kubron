import type { ColumnDef } from '../types'

// Элементы Role/RoleBinding/... приходят в k8s-формате (metadata.name)
export const nameCol: ColumnDef = { key: 'metadata.name', label: 'Name', className: 'resource-name' }
export const nsCol: ColumnDef = { key: 'metadata.namespace', label: 'Namespace', className: 'ns-label' }

// Остальные приходят плоскими (name, namespace)
export const flatNameCol: ColumnDef = { key: 'name', label: 'Name', className: 'resource-name' }
export const flatNsCol: ColumnDef = { key: 'namespace', label: 'Namespace', className: 'ns-label' }

export const ageCol: ColumnDef = { key: 'age', label: 'Age' }
