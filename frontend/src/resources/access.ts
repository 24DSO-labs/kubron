import { can } from '../composables/useAuth'
import { Perm, type Permission } from '../types/auth'
import type { ResourceDefinition } from './types'
import { hasTemplate } from './templates'

/**
 * Видимость элементов интерфейса по правам. Это только удобство: бэкенд проверяет права сам.
 * Функции читают реактивное состояние, поэтому годятся для computed и шаблонов.
 */

/** Можно ли показать действие (по умолчанию для действия нужно право на запись) */
export const actionAllowed = (a: { permission?: Permission }) => can(a.permission ?? Perm.Write)

/** Можно ли удалять ресурсы этого типа */
export const canDelete = (def: ResourceDefinition) => !!def.api.remove && can(Perm.Write)

/** Можно ли менять YAML (иначе YAML открывается только для чтения) */
export const canEditYaml = () => can(Perm.Write)

/** Показывать ли чекбоксы и кнопку «−» (массовое удаление) */
export const canBulkDelete = (def: ResourceDefinition) => canDelete(def) && def.bulkDelete !== false

/** Показывать ли кнопку «+»: для типа есть базовый манифест, и есть право на запись */
export const canCreate = (def: ResourceDefinition) => def.editable !== false && hasTemplate(def) && can(Perm.Write)
