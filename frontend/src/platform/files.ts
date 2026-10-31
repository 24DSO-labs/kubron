// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { BrowserOpenURL } from '../../wailsjs/runtime/runtime'
import { t } from '../i18n'

/**
 * Открывает ссылку в браузере пользователя.
 * Десктоп: системный браузер через Wails. Веб: window.open (этап 1).
 */
export function openUrl(url: string): void {
  if (import.meta.env.VITE_TARGET === 'web') {
    window.open(url, '_blank', 'noopener,noreferrer')
    return
  }
  BrowserOpenURL(url)
}

/**
 * Выбор текстового файла в браузере (<input type=file>) и чтение его содержимого.
 * Нужен вебу, где у сервера нет доступа к файлам пользователя (capabilities.fileDialog = false):
 * kubeconfig передаётся текстом. Возвращает null, если выбор отменён.
 */
export function pickTextFile(accept = '', maxBytes = 1 << 20): Promise<string | null> {
  return new Promise((resolve, reject) => {
    const input = document.createElement('input')
    input.type = 'file'
    if (accept) input.accept = accept
    input.onchange = () => {
      const file = input.files?.[0]
      if (!file) return resolve(null)
      if (file.size > maxBytes) return reject(new Error(t('platform.fileTooLarge', { kb: Math.round(maxBytes / 1024) })))
      file.text().then(resolve, reject)
    }
    input.oncancel = () => resolve(null)
    input.click()
  })
}

/**
 * Сохраняет текст файлом через браузер (ссылка на Blob). Нужен вебу; в десктопе (Wails) такая загрузка не
 * показывает диалог сохранения, поэтому там выгрузки не предлагаются (см. capabilities.teams).
 */
export function downloadTextFile(name: string, content: string, mime = 'text/plain'): void {
  const url = URL.createObjectURL(new Blob([content], { type: `${mime};charset=utf-8` }))
  const a = document.createElement('a')
  a.href = url
  a.download = name
  document.body.appendChild(a)
  a.click()
  a.remove()
  setTimeout(() => URL.revokeObjectURL(url), 1000)
}
