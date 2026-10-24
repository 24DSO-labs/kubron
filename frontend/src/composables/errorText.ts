/** Текст ошибки из чего угодно: Wails отвечает строкой, остальной код — Error. */
export function errorText(e: unknown): string {
  if (e instanceof Error) return e.message
  if (typeof e === 'string') return e
  return String(e)
}
