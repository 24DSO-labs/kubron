/** Цвета и css-классы статусов: одно место для списков и панелей */

const OK = ['running', 'ready', 'active', 'bound', 'succeeded', 'completed', 'complete', 'available', 'true', 'deployed']
const WARN = ['pending', 'containercreating', 'podinitializing', 'terminating', 'progressing', 'pending-install', 'pending-upgrade', 'pending-rollback', 'pending-delete', 'uninstalling', 'superseded']
const ERR = [
  'failed', 'error', 'crashloopbackoff', 'crashed', 'imagepullbackoff', 'errimagepull', 'evicted',
  'lost', 'false', 'replicafailure', 'progressdeadlineexceeded',
]

/** css-класс бейджа (стили в components/resource/resource.css) */
export function badgeClass(status?: string): string {
  const s = (status ?? '').toLowerCase()
  if (OK.includes(s)) return 'rd-ok'
  if (WARN.includes(s)) return 'rd-warn'
  if (ERR.includes(s)) return 'rd-err'
  return 'rd-unk'
}

/** Цвет текста для колонок списка (colorFn) */
export function statusColor(status?: string): string {
  switch (badgeClass(status)) {
    case 'rd-ok': return '#4ade80'
    case 'rd-warn': return '#facc15'
    case 'rd-err': return '#ef4444'
    default: return '#868d8a'
  }
}
