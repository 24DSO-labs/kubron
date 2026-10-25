/**
 * Единственное место, где мы обращаемся к Go-биндингам по имени (десктоп, Wails).
 * Веб-реализация (POST /api/rpc/{method}) появится в этапе 1 и выберется при сборке (ADR-0010).
 *
 * Имя метода и аргументы проверяются по wailsjs/go/main/App.d.ts: опечатка в имени
 * или неверное число/тип аргументов ловится в `vue-tsc`, а не в рантайме.
 * Динамический импорт и проверка наличия функции оставлены на случай устаревших биндингов.
 */
type AppApi = typeof import('../../wailsjs/go/main/App')

/** Имена методов Go-приложения, доступные из фронтенда */
export type AppMethod = {
  [K in keyof AppApi]: AppApi[K] extends (...args: any[]) => Promise<any> ? K : never
}[keyof AppApi]

/** Текст ошибки бэкенда «нет сессии» (auth.ErrNotAuthenticated) */
export const NOT_AUTHENTICATED = 'требуется вход в систему'

/** Методы, для которых «нет сессии» — штатный ответ (до входа), а не признак истёкшей сессии */
const PRE_LOGIN_METHODS = new Set<string>(['AuthStatus', 'SetupAdmin', 'Login', 'Logout', 'CurrentUser'])

let sessionLostHandler: (() => void) | null = null

/** Обработчик потери сессии на стороне бэкенда (автовыход по бездействию и т.п.) */
export function onSessionLost(handler: () => void) {
  sessionLostHandler = handler
}

export async function callApp<K extends AppMethod>(
  fn: K,
  ...args: Parameters<AppApi[K]>
): Promise<Awaited<ReturnType<AppApi[K]>>> {
  const app: any = await import('../../wailsjs/go/main/App')
  if (typeof app[fn] !== 'function') {
    throw new Error(`${String(fn)} недоступен в Go-биндингах`)
  }
  try {
    return await app[fn](...args)
  } catch (e) {
    const text = typeof e === 'string' ? e : (e as { message?: string } | null)?.message ?? ''
    if (text.includes(NOT_AUTHENTICATED) && !PRE_LOGIN_METHODS.has(String(fn))) sessionLostHandler?.()
    throw e
  }
}
