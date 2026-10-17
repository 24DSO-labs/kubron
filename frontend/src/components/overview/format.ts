/** Форматирование значений для главного экрана */

const GIB = 1024 ** 3
const MIB = 1024 ** 2

export function formatBytes(bytes: number): string {
  if (!isFinite(bytes) || bytes <= 0) return '0 GiB'
  if (bytes >= GIB) return `${(bytes / GIB).toFixed(bytes >= 100 * GIB ? 0 : 1)} GiB`
  return `${(bytes / MIB).toFixed(0)} MiB`
}

export function formatCores(cores: number): string {
  if (!isFinite(cores) || cores <= 0) return '0'
  return cores >= 10 ? cores.toFixed(0) : cores.toFixed(1)
}

/** «5m ago», «2h ago», «3d ago» по RFC3339-времени */
export function timeAgo(iso: string, now: number = Date.now()): string {
  const t = Date.parse(iso)
  if (isNaN(t)) return ''
  const sec = Math.max(0, Math.round((now - t) / 1000))
  if (sec < 60) return 'just now'
  const min = Math.floor(sec / 60)
  if (min < 60) return `${min}m ago`
  const h = Math.floor(min / 60)
  if (h < 24) return `${h}h ago`
  return `${Math.floor(h / 24)}d ago`
}
