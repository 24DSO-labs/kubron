// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import type { ClusterHealth, ClusterStatus } from '../types'
import { t } from '../i18n'

/** ok — отвечает; bad — недоступен или отклонил учётные данные; unknown — не проверялся (десктоп) */
export type HealthLevel = 'ok' | 'bad' | 'unknown'

const BAD_STATES = new Set(['unreachable', 'unauthorized', 'misconfigured'])

export function healthLevel(health: ClusterHealth | undefined | null): HealthLevel {
  if (!health) return 'unknown'
  if (health.state === 'ok') return 'ok'
  return BAD_STATES.has(health.state) ? 'bad' : 'unknown'
}

/** Подпись состояния для подсказки: «Недоступен: dial tcp …». Для исправного кластера и без проверки — пусто. */
export function healthText(health: ClusterHealth | undefined | null): string {
  if (healthLevel(health) !== 'bad' || !health) return ''
  const label = t(`cluster.health.${health.state}`)
  return health.message ? `${label}: ${health.message}` : label
}

/** Подсказка к значку кластера в полосе: имя, окружение и, если есть, причина недоступности. */
export function clusterTitle(base: string, cluster: Pick<ClusterStatus, 'health'>): string {
  const text = healthText(cluster.health)
  return text ? `${base}\n${text}` : base
}
