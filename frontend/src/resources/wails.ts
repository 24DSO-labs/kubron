/**
 * Единственное место, где мы обращаемся к Go-биндингам по имени.
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

export async function callApp<K extends AppMethod>(
  fn: K,
  ...args: Parameters<AppApi[K]>
): Promise<Awaited<ReturnType<AppApi[K]>>> {
  const app: any = await import('../../wailsjs/go/main/App')
  if (typeof app[fn] !== 'function') {
    throw new Error(`${String(fn)} недоступен в Go-биндингах`)
  }
  return app[fn](...args)
}
