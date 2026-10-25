import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'

/**
 * Открывает ссылку в браузере пользователя.
 * Десктоп: системный браузер через Wails. Веб: window.open (этап 1).
 */
export function openUrl(url: string): void {
  BrowserOpenURL(url)
}
