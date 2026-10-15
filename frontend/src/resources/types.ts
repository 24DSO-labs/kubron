import type { Component } from 'vue'

/** Колонка списка: повторяет Column из K8sResourceListBase_old */
export interface ColumnDef {
  key: string // dotted-path: 'metadata.name'
  label: string
  className?: string
  component?: Component // кастомная ячейка (props: item, col)
  formatter?: (item: any) => string // вычисляемое значение
  colorFn?: (item: any) => string // цвет значения
}

export interface MenuItemDef {
  label: string
  action: string
  danger?: boolean
}

/** Единая точка действий. Реализация: resources/actions.ts */
export interface ActionContext {
  open(id: string, ns: string, name: string, item?: any): void
  edit(id: string, ns: string, name: string): Promise<void>
  requestDelete(id: string, ns: string, name: string): void
  openLogs(ns: string, pod: string, containers?: { name: string }[], container?: string): void
  /** Логи workload-ресурса (deployment, daemonset): поды выбирает TabbedPanel */
  openResourceLogs(kind: string, ns: string, name: string): void
  openShell(ns: string, pod: string, container: string): Promise<void>
  openAttach(ns: string, pod: string, container: string): Promise<void>
  /** Открыть диалог масштабирования Deployment */
  requestScale(item: any): void
  /** Подтверждение (пока window.confirm; заменить на общий диалог в одном месте) */
  confirm(message: string): boolean
  /** try/catch + тосты + обновление списка. Бросать исключение внутри fn безопасно */
  perform(fn: () => Promise<unknown>, successMessage?: string): Promise<void>
  reload(): Promise<void>
  notifySuccess(msg: string): void
  notifyError(msg: string): void
}

export interface PropertyDef<T = any> {
  label: string
  value: (item: T) => string | number | null | undefined
  /** Показать значение бейджем: функция возвращает css-класс (см. resources/status.ts) */
  badge?: (item: T) => string
}

export interface ActionDef<T = any> {
  id: string // 'logs' | 'shell' | 'scale' ...
  label: string
  /** URL svg-иконки для кнопки в шапке панели; без иконки рисуется текст */
  icon?: string
  danger?: boolean
  /** Видимость кнопки в шапке панели деталей */
  when?: (item: T) => boolean
  run: (item: T, ctx: ActionContext) => void | Promise<void>
}

export interface TabDef {
  id: string
  label: string
  /** Получает prop `resource`; событие `open` ({ id, ns, name, item? }) открывает другой ресурс */
  component: Component
}

export interface ResourceDefinition<T = any> {
  /** lowercase-тип; он же `kind` для GetResourceYAML: 'role', 'pod' ... */
  id: string
  /** Отображаемое имя типа: 'Role' */
  kind: string
  /** id во view / в Sidebar: 'roles' */
  view: string
  /** Заголовок списка */
  title: string
  icon?: string
  namespaced: boolean

  columns: ColumnDef[]
  /**
   * Панель деталей, вкладка Properties. Порядок:
   * headSections -> Labels -> Annotations -> properties -> sections
   */
  headSections?: Component[]
  properties?: PropertyDef<T>[]
  sections?: Component[]
  /** Дополнительные вкладки панели (кроме Properties и Events) */
  tabs?: TabDef[]
  /** Специфичные действия; Edit YAML и Delete добавляются автоматически */
  actions?: ActionDef<T>[]

  api: {
    list: (ns: string) => Promise<T[]>
    /** Свежие данные при открытии панели (богаче, чем элемент списка) */
    get?: (ns: string, name: string) => Promise<T>
    remove?: (ns: string, name: string) => Promise<void>
  }
}

export const defineResource = <T = any>(d: ResourceDefinition<T>) => d
