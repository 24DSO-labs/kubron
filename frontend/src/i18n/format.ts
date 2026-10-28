// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

/**
 * Даты и числа в формате текущего языка интерфейса (а не системного или зашитого 'ru-RU').
 * Читают реактивный `locale`, поэтому вызов внутри computed/render пересчитывается при смене языка.
 */
import { locale } from './index'

type DateInput = Date | string | number

function toDate(value: DateInput): Date {
  return value instanceof Date ? value : new Date(value)
}

/** Невалидная дата → '' (вызывающий подставляет свой «—»), а не «Invalid Date» */
function format(value: DateInput, options: Intl.DateTimeFormatOptions): string {
  const date = toDate(value)
  if (Number.isNaN(date.getTime())) return ''
  return new Intl.DateTimeFormat(locale.value, options).format(date)
}

export const formatDateTime = (value: DateInput) => format(value, { dateStyle: 'medium', timeStyle: 'medium' })
export const formatDate = (value: DateInput) => format(value, { dateStyle: 'medium' })
export const formatTime = (value: DateInput) => format(value, { timeStyle: 'medium' })

/** Короткая подпись оси времени: «14:05» */
export const formatClock = (value: DateInput) => format(value, { hour: '2-digit', minute: '2-digit' })

/** Короткая подпись оси дат: «26 окт.» / «Oct 26» */
export const formatMonthDay = (value: DateInput) => format(value, { month: 'short', day: 'numeric' })

export const formatNumber = (value: number, options?: Intl.NumberFormatOptions) =>
  new Intl.NumberFormat(locale.value, options).format(value)
