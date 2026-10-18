import { ref } from 'vue'

export type NotificationType = 'error' | 'success' | 'info'

export interface Notification {
  id: number
  type: NotificationType
  message: string
}

let nextId = 0

// Одно хранилище на всё приложение: App.vue рисует тосты, а уведомления шлют и composables,
// и resources/actions.ts. Если держать список внутри функции, у каждого вызова был бы свой.
const notifications = ref<Notification[]>([])

export function useNotifications() {
  function notify(message: string, type: NotificationType = 'info', duration = 4000) {
    const id = nextId++
    notifications.value.push({ id, type, message })
    if (duration > 0) {
      setTimeout(() => dismiss(id), duration)
    }
  }

  function dismiss(id: number) {
    notifications.value = notifications.value.filter(n => n.id !== id)
  }

  function notifyError(message: string) {
    notify(message, 'error', 6000)
  }

  function notifySuccess(message: string) {
    notify(message, 'success', 3000)
  }

  return { notifications, notify, dismiss, notifyError, notifySuccess }
}