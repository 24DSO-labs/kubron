import { defineAsyncComponent, type Component } from 'vue'

export interface ViewConfig {
  component: Component
}

// Ленивая загрузка для тяжёлых/редких компонентов
const lazy = (path: string) => defineAsyncComponent(() => import(`../components/views/${path}.vue`))

/*
 * Кастомные view, не вписывающиеся в каркас ресурсов (список + панель деталей):
 *  - overview — дашборд кластера с графиками
 *  - settings — форма настроек кластера
 *  - users — управление пользователями и ролями (только для users.manage)
 *  - audit — журнал аудита (только для audit.read)
 * Все остальные view (k8s-ресурсы, helm, port forwarding) — data-driven, см. resources/registry.ts
 */
export const viewRegistry: Record<string, ViewConfig> = {
  overview: {
    component: lazy('OverviewView'),
  },
  settings: {
    component: lazy('ClusterSettings'),
  },
  // Управление пользователями; бэкенд всё равно требует право users.manage
  users: {
    component: lazy('UsersView'),
  },
  // Журнал аудита; бэкенд всё равно требует право audit.read
  audit: {
    component: lazy('AuditView'),
  },
}
