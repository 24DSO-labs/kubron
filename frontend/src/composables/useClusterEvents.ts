// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import type { Ref } from 'vue'
import type { ClusterStatus } from '../types'
import { onEvent } from '../platform/events'
import { t } from '../i18n'

/** Состояние соединения какого-то кластера изменилось: список перечитывается. */
export const EVENT_CLUSTERS_HEALTH = 'clusters:health'
/** Пользователь потерял доступ к активному кластеру: сервер закрыл его терминалы, логи и port-forward. */
export const EVENT_CLUSTER_ACCESS_REVOKED = 'cluster:access-revoked'

export interface ClusterEventDeps {
  clusters: Ref<ClusterStatus[]>
  activeClusterId: Ref<string>
  loadClusters: () => Promise<void>
  /** Сброс активного кластера на стороне интерфейса (то же, что отключение). */
  disconnect: () => void
  notifyError: (message: string) => void
}

/** Обработчик потери доступа; вынесен отдельно ради тестов. Имя берётся до перечитывания списка. */
export function handleAccessRevoked(clusterId: unknown, deps: ClusterEventDeps) {
  const id = typeof clusterId === 'string' ? clusterId : ''
  const name = deps.clusters.value.find(c => c.id === id)?.displayName || id
  if (id && deps.activeClusterId.value === id) deps.disconnect()
  void deps.loadClusters()
  deps.notifyError(t('cluster.accessRevoked', { name }))
}

/** Подписка на события кластеров; вызывать один раз при старте. Возвращает функцию отписки. */
export function listenClusterEvents(deps: ClusterEventDeps): () => void {
  const stops = [
    onEvent(EVENT_CLUSTERS_HEALTH, () => { void deps.loadClusters() }),
    onEvent(EVENT_CLUSTER_ACCESS_REVOKED, (id: unknown) => handleAccessRevoked(id, deps)),
  ]
  return () => stops.forEach(stop => stop())
}
