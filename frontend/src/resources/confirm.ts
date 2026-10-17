import { shallowRef } from 'vue'

/**
 * Диалог подтверждения действий (Restart, Evict, Rollback ...).
 * Отдельный модуль без зависимостей: его можно импортировать и из actions.ts, и из секций
 * (секции не импортируют actions.ts: registry -> definitions -> секции дал бы цикл).
 * Сам диалог рисует ResourceHost.
 */
export const pendingConfirm = shallowRef<{ message: string; resolve: (ok: boolean) => void } | null>(null)

/** true — «Подтвердить», false — отмена (или новый запрос вытеснил этот) */
export function askConfirm(message: string): Promise<boolean> {
  pendingConfirm.value?.resolve(false)
  return new Promise<boolean>(resolve => {
    pendingConfirm.value = { message, resolve }
  })
}

export function answerConfirm(ok: boolean): void {
  const p = pendingConfirm.value
  pendingConfirm.value = null
  p?.resolve(ok)
}
