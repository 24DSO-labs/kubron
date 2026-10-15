/**
 * Единственное место, где мы обращаемся к Go-биндингам по имени.
 * Поведение совпадает с текущим useK8s (динамический импорт + проверка наличия),
 * но когда биндинги стабилизируются, можно заменить на статические импорты
 * прямо здесь, не трогая определения ресурсов.
 */
export async function callApp<T = any>(fn: string, ...args: any[]): Promise<T> {
  const app: any = await import('../../wailsjs/go/main/App')
  if (typeof app[fn] !== 'function') {
    throw new Error(`${fn} недоступен в Go-биндингах`)
  }
  return app[fn](...args)
}
