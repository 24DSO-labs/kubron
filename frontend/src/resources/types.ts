import type { Component } from 'vue'

/**
 * Wails генерирует классы с методами (convertValues), а по сети приходят обычные JSON-объекты.
 * Item<T> оставляет только данные, поэтому `{ ...pod, cpu }` остаётся совместимым с типом.
 * Для T = any (ресурсы без модели: Role, ClusterRole ...) остаётся any.
 */
export type Item<T> = 0 extends 1 & T
  ? any
  : { [K in keyof T as T[K] extends (...a: any[]) => any ? never : K]: T[K] }

/**
 * Элемент в панели деталей: поля элемента списка (T) плюс то, что добавляет api.get (D).
 * Из контекстного меню действие получает элемент списка, поэтому поля D необязательные.
 */
export type PanelItem<T, D> = Item<T> & Partial<Item<D>>

/** Колонка списка */
export interface ColumnDef<T = any> {
  key: string // dotted-path: 'metadata.name'
  label: string
  className?: string
  component?: Component // кастомная ячейка (props: item, col)
  formatter?: (item: T) => string // вычисляемое значение
  colorFn?: (item: T) => string // цвет значения
}

/** Группы бокового меню; заголовки и порядок — в resources/navigation.ts */
export type NavGroupId = 'workloads' | 'config' | 'network' | 'storage' | 'helm' | 'access'

/** Пункт бокового меню. Без `nav` ресурс в меню не показывается */
export interface ResourceNav {
  /** Без group пункт стоит на верхнем уровне */
  group?: NavGroupId
  label: string
}

export interface MenuItemDef {
  label: string
  action: string
  danger?: boolean
  /** Подменю (например, контейнеры пода); выбранный arg попадёт в ActionDef.run третьим аргументом */
  submenu?: { label: string; arg: string }[]
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
  /** Диалог подтверждения (рисует ResourceHost). Резолвится true по «Подтвердить», false по отмене */
  confirm(message: string): Promise<boolean>
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
  /** Ссылка на другой ресурс: 'pod', 'namespace', 'node' ... */
  link?: (item: T) => { id: string; ns: string; name: string } | null | undefined
  /** Текст ссылки (если отличается от value). value рендерится как обычный текст, linkLabel — как ссылка */
  linkLabel?: (item: T) => string | null | undefined
}

export interface ActionDef<T = any> {
  id: string // 'logs' | 'shell' | 'scale' ...
  label: string
  /** URL svg-иконки для кнопки в шапке панели; без иконки рисуется текст */
  icon?: string
  danger?: boolean
  /** Видимость кнопки в шапке панели деталей */
  when?: (item: T) => boolean
  /**
   * Действие над контейнером пода (attach/shell/logs): если у ресурса больше одного контейнера,
   * панель показывает выпадающий список и передаёт выбранный контейнер третьим аргументом run.
   * Контейнеры берутся из item.containers.
   */
  perContainer?: boolean
  run: (item: T, ctx: ActionContext, container?: string) => void | Promise<void>
}

export interface TabDef {
  id: string
  label: string
  /** Получает prop `resource`; событие `open` ({ id, ns, name, item? }) открывает другой ресурс */
  component: Component
}

/**
 * T — элемент списка (api.list), D — что добавляет api.get (панель деталей).
 * Типы берём из wailsjs: defineResource<k8s.PodInfo, k8s.PodDetail>(...).
 * Без параметров (T = any) проверки типов нет.
 */
export interface ResourceDefinition<T = any, D = {}> {
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
  /** Пункт в Sidebar; порядок внутри группы — порядок в resources/registry.ts */
  nav?: ResourceNav

  columns: ColumnDef<Item<T>>[]
  /**
   * Панель деталей, вкладка Properties. Порядок:
   * headSections -> Labels -> Annotations -> properties -> sections
   */
  headSections?: Component[]
  properties?: PropertyDef<PanelItem<T, D>>[]
  sections?: Component[]
  /** Дополнительные вкладки панели (кроме Properties и Events) */
  tabs?: TabDef[]
  /** Специфичные действия; Edit YAML и Delete добавляются автоматически */
  actions?: ActionDef<PanelItem<T, D>>[]

  /**
   * Есть ли у ресурса YAML в кластере (Edit YAML).
   * false — для сущностей вне k8s API: helm-релизы, port forward сессии.
   */
  editable?: boolean

  /**
   * Реактивный источник, при изменении которого открытый список перезагружается
   * (данные меняются не только из списка: например, port forward запускают из панели pod/service).
   * Функция читает ref'ы: `refreshOn: () => k8s.portForwards.value`.
   */
  refreshOn?: () => unknown

  /**
   * Вызывается после успешной загрузки списка (например, для метрик pod).
   * Если вернуть массив, он заменит список в сторе: данные не мутируем на месте,
   * иначе shallowRef не заметит изменений и таблица не перерисуется.
   */
  onLoaded?: (items: Item<T>[]) => void | Item<T>[] | Promise<void | Item<T>[]>

  api: {
    list: (ns: string) => Promise<T[]>
    /** Свежие данные при открытии панели (богаче, чем элемент списка) */
    get?: (ns: string, name: string) => Promise<D>
    remove?: (ns: string, name: string) => Promise<void>
  }
}

export const defineResource = <T = any, D = {}>(d: ResourceDefinition<T, D>) => d
