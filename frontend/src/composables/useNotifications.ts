import { ref } from 'vue'

export type NotificationType = 'error' | 'success' | 'info'

export interface Notification {
  id: number
  type: NotificationType
  message: string
}

let nextId = 0

export function useNotifications() {
  const notifications = ref<Notification[]>([])

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