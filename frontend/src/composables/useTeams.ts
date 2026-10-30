// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { computed, ref } from 'vue'
import type { Team } from '../types'
import { callApp } from '../platform/api'
import { errorText } from './errorText'

/**
 * Команды текущего пользователя (администратор организации видит все). Состояние одно на приложение.
 * Это удобство интерфейса: права проверяет бэкенд в каждом методе.
 */
const teams = ref<Team[]>([])
const loading = ref(false)
const loadError = ref('')

async function loadTeams(): Promise<void> {
  loading.value = true
  loadError.value = ''
  try {
    teams.value = ((await callApp('ListTeams')) ?? []) as Team[]
  } catch (e) {
    teams.value = []
    loadError.value = errorText(e)
  } finally {
    loading.value = false
  }
}

/** Команды, в которые пользователь может добавлять кластеры (он их администратор) */
const adminTeams = computed(() => teams.value.filter(t => t.myRole === 'admin'))

export function useTeams() {
  return { teams, adminTeams, loading, loadError, loadTeams }
}
