/** Результат действия; значения совпадают с internal/audit */
export type AuditResult = 'ok' | 'failed' | 'denied'

/** Запись журнала (internal/audit.Event) */
export interface AuditEvent {
  time: string
  user: string
  role?: string
  action: string
  result: AuditResult | string
  detail?: string
}

/** Условия выборки (internal/audit.Filter) */
export interface AuditFilter {
  user: string
  action: string
  text: string
  result: string
  since: string
  until: string
  limit: number
}

/** Ответ GetAuditLog (internal/audit.Page) */
export interface AuditPage {
  events: AuditEvent[]
  truncated: boolean
  writeError: string
}

export const RESULT_LABELS: Record<string, string> = {
  ok: 'Выполнено',
  failed: 'Ошибка',
  denied: 'Отказано',
}

const ACTION_LABELS: Record<string, string> = {
  Setup: 'Первичная настройка',
  Login: 'Вход',
  Logout: 'Выход',
  SessionTimeout: 'Автовыход по бездействию',
  SetIdleTimeout: 'Настройка автовыхода',
  ChangePassword: 'Смена пароля',
  CreateUser: 'Создание пользователя',
  UpdateUserRole: 'Смена роли',
  SetUserDisabled: 'Блокировка / разблокировка',
  ResetUserPassword: 'Сброс пароля',
  DeleteUser: 'Удаление пользователя',
  ApplyYAML: 'Применение YAML',
  DetectShell: 'Определение shell в поде',
  StartTerminalSession: 'Открытие терминала',
  CloseTerminal: 'Закрытие терминала',
  GetSecretValue: 'Просмотр значения секрета',
  StartPortForward: 'Проброс порта (под)',
  StartServicePortForward: 'Проброс порта (сервис)',
  StopPortForward: 'Остановка проброса порта',
  AddClusterFromFile: 'Добавление кластера (файл)',
  AddClusterFromText: 'Добавление кластера (текст)',
  UpdateCluster: 'Изменение кластера',
  DeleteCluster: 'Удаление кластера',
  DeleteResource: 'Удаление ресурса',
  InstallMetricsStack: 'Установка стека метрик',
  UninstallMetricsStack: 'Удаление стека метрик',
  GetAuditLog: 'Просмотр журнала',
  SetDefaultIngressClass: 'Класс Ingress по умолчанию',
  SyncArgoApplication: 'Argo CD: синхронизация',
  RefreshArgoApplication: 'Argo CD: обновление (Refresh)',
  TerminateArgoOperation: 'Argo CD: остановка операции',
  RollbackArgoApplication: 'Argo CD: откат',
  OpenArgoUI: 'Argo CD: открытие UI',
  SetArgoAPIToken: 'Argo CD: сохранение токена API',
  GenerateArgoAPIToken: 'Argo CD: выдача токена API',
  DeleteArgoAPIToken: 'Argo CD: удаление токена API',
  GetArgoApplicationDiff: 'Argo CD: сравнение с git (diff)',
  SuspendArgoAutoSync: 'Argo CD: отключение автосинхронизации',
  ResumeArgoAutoSync: 'Argo CD: включение автосинхронизации',
  DeleteArgoApplication: 'Argo CD: удаление приложения',
  GetHelmReleaseContent: 'Helm: просмотр values / манифеста релиза',
  RollbackHelmRelease: 'Helm: откат релиза',
  UninstallHelmRelease: 'Helm: удаление релиза',
  GetHelmRevisionDiff: 'Helm: сравнение ревизий',

}  

const ACTION_VERBS: [string, string][] = [
  ['Delete', 'Удаление'],
  ['Scale', 'Масштабирование'],
  ['Restart', 'Перезапуск'],
  ['Rollback', 'Откат'],
  ['Retry', 'Повтор'],
  ['Evict', 'Выселение'],
]

/** Понятное название действия; неизвестные имена показываются как есть */
export function actionLabel(action: string): string {
  const exact = ACTION_LABELS[action]
  if (exact) return exact
  for (const [prefix, verb] of ACTION_VERBS) {
    if (action.startsWith(prefix) && action.length > prefix.length) return `${verb}: ${action.slice(prefix.length)}`
  }
  return action
}
