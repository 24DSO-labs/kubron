/** Результат пакетной операции: исключение одного элемента не прерывает остальные */
export interface BatchResult<T> {
  done: T[]
  failed: { item: T; error: string }[]
}

const errText = (e: unknown): string =>
  (e as { message?: string } | null)?.message || (typeof e === 'string' ? e : String(e))

/**
 * Выполняет fn для каждого элемента не более чем `limit` вызовов одновременно.
 * Не бросает исключений: ошибки собираются в `failed`. Порядок `done` и `failed` — порядок завершения.
 */
export async function runBatch<T>(items: readonly T[], fn: (item: T) => Promise<unknown>, limit = 4): Promise<BatchResult<T>> {
  const result: BatchResult<T> = { done: [], failed: [] }
  let next = 0

  async function worker() {
    while (next < items.length) {
      const item = items[next++]
      try {
        await fn(item)
        result.done.push(item)
      } catch (e) {
        result.failed.push({ item, error: errText(e) })
      }
    }
  }

  const workers = Math.max(1, Math.min(limit, items.length))
  await Promise.all(Array.from({ length: workers }, worker))
  return result
}
