/**
 * Права. Значения должны совпадать с internal/auth/permissions.go.
 * Интерфейс проверяет именно права (can), а не роли: новая роль на бэкенде
 * не требует правок на фронтенде.
 */
export const Perm = {
  Read: 'read',
  SecretsRead: 'secrets.read',
  Write: 'write',
  Exec: 'exec',
  PortForward: 'portforward',
  ClusterManage: 'clusters.manage',
  UsersManage: 'users.manage',
} as const

export type Permission = (typeof Perm)[keyof typeof Perm]

/** Пользователь из Go (internal/auth.UserInfo) */
export interface UserInfo {
  id: string
  username: string
  role: string
  roleLabel: string
  permissions: string[]
  disabled: boolean
  mustChangePassword: boolean
  createdAt: string
  lastLoginAt: string
}

/** Роль из Go (internal/auth.RoleInfo) */
export interface RoleInfo {
  id: string
  label: string
  permissions: string[]
}

/** Подписи прав для интерфейса управления пользователями */
export const PERMISSION_LABELS: Record<string, string> = {
  [Perm.Read]: 'Просмотр ресурсов, логов, метрик, YAML',
  [Perm.SecretsRead]: 'Просмотр значений секретов',
  [Perm.Write]: 'Изменение и удаление ресурсов',
  [Perm.Exec]: 'Терминал в подах',
  [Perm.PortForward]: 'Проброс портов',
  [Perm.ClusterManage]: 'Управление кластерами',
  [Perm.UsersManage]: 'Управление пользователями',
}
