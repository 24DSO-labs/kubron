// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/** Форматирование значений для главного экрана */
import { t } from '../../i18n'
import { formatNumber } from '../../i18n/format'

const GIB = 1024 ** 3
const MIB = 1024 ** 2

/** Число с фиксированным числом знаков в формате текущего языка */
const fixed = (v: number, digits: number) =>
  formatNumber(v, { minimumFractionDigits: digits, maximumFractionDigits: digits })

export function formatBytes(bytes: number): string {
  if (!isFinite(bytes) || bytes <= 0) return '0 GiB'
  if (bytes >= GIB) return `${fixed(bytes / GIB, bytes >= 100 * GIB ? 0 : 1)} GiB`
  return `${fixed(bytes / MIB, 0)} MiB`
}

export function formatCores(cores: number): string {
  if (!isFinite(cores) || cores <= 0) return '0'
  return fixed(cores, cores >= 10 ? 0 : 1)
}

/** «2.5 cores» */
export const formatCoresLabel = (cores: number): string => t('overview.cores', { value: formatCores(cores) })

/** «5m ago», «2h ago», «3d ago» по RFC3339-времени */
export function timeAgo(iso: string, now: number = Date.now()): string {
  const ts = Date.parse(iso)
  if (isNaN(ts)) return ''
  const sec = Math.max(0, Math.round((now - ts) / 1000))
  if (sec < 60) return t('overview.time.justNow')
  const min = Math.floor(sec / 60)
  if (min < 60) return t('overview.time.minutesAgo', { n: min })
  const h = Math.floor(min / 60)
  if (h < 24) return t('overview.time.hoursAgo', { n: h })
  return t('overview.time.daysAgo', { n: Math.floor(h / 24) })
}
