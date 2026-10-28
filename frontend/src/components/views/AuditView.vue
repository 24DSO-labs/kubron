<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="audit-view">
    <header class="audit-head">
      <div>
        <h2>{{ $t('audit.title') }}</h2>
        <p class="audit-sub">{{ $t('audit.subtitle') }}</p>
      </div>
      <button class="btn" type="button" :disabled="loading" @click="load">{{ $t('audit.refresh') }}</button>
    </header>

    <p v-if="page?.writeError" class="audit-warn">
      {{ $t('audit.writeError', { error: page.writeError }) }}
    </p>

    <form class="audit-filters" @submit.prevent="load">
      <input v-model="filter.user" class="form-input" type="search" 
        :placeholder="$t('audit.filters.user')" :aria-label="$t('audit.filters.user')" />
      <input v-model="filter.action" class="form-input" type="search" :placeholder="$t('audit.filters.actionPlaceholder')"
        :aria-label="$t('audit.filters.action')" />
      <input v-model="filter.text" class="form-input" type="search" :placeholder="$t('audit.filters.objectPlaceholder')"
        :aria-label="$t('audit.filters.object')" />
      <select v-model="filter.result" class="form-input" 
        :aria-label="$t('audit.filters.result')">
        <option value="">{{ $t('audit.filters.anyResult') }}</option>
        <option v-for="id in RESULT_IDS" :key="id" :value="id">{{ resultLabel(id) }}</option>
      </select>
      <select v-model="period" class="form-input" 
        :aria-label="$t('audit.filters.period')">
        <option v-for="p in PERIODS" :key="p.id" :value="p.id">{{ $t(`audit.periods.${p.id}`) }}</option>
      </select>
      <select v-model.number="filter.limit" class="form-input" 
        :aria-label="$t('audit.filters.limit')">
        <option v-for="n in LIMITS" :key="n" :value="n">{{ $t('audit.filters.show', { n }) }}</option>
      </select>
    </form>

    <p v-if="loadError" class="form-error">{{ loadError }}</p>

    <p v-if="loading && !page" class="audit-empty">{{ $t('audit.loading') }}</p>
    <p v-else-if="page && !page.events.length && !loadError" class="audit-empty">{{ $t('audit.empty') }}</p>

    <table v-if="page?.events.length" class="audit-table">
      <thead>
        <tr>
          <th>{{ $t('audit.columns.time') }}</th>
          <th>{{ $t('audit.columns.user') }}</th>
          <th>{{ $t('audit.columns.action') }}</th>
          <th>{{ $t('audit.columns.result') }}</th>
          <th>{{ $t('audit.columns.detail') }}</th>
        </tr>
      </thead>
      <tbody>
        <tr v-for="(e, i) in page.events" :key="i" :class="{ denied: e.result === 'denied' }">
          <td class="nowrap muted">{{ formatTime(e.time) }}</td>
          <td class="nowrap">
            <template v-if="e.user">
              <strong>{{ e.user }}</strong>
              <span v-if="e.role" class="role-tag">{{ roleLabel(e.role) }}</span>
            </template>
            <span v-else class="muted">{{ $t('common.dash') }}</span>
          </td>
          <td :title="e.action">{{ actionLabel(e.action) }}</td>
          <td>
            <span :class="['badge', `badge-${e.result}`]">{{ resultLabel(e.result) }}</span>
          </td>
          <td class="detail">{{ e.detail }}</td>
        </tr>
      </tbody>
    </table>

    <p v-if="page?.truncated" class="audit-more">
      {{ $t('audit.truncated', { n: page.events.length }) }}
    </p>
  </div>
</template>

<script lang="ts" setup>
import { onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue'
import { callApp } from '../../resources/wails'
import { errorText } from '../../composables/errorText'
import { formatDateTime } from '../../i18n/format'
import { te, t } from '../../i18n'
import { RESULT_IDS, resultLabel, actionLabel, type AuditFilter, type AuditPage } from '../../types/audit'

const HOUR = 3600_000
const PERIODS = [
  { id: 'all', ms: 0 },
  { id: '1h', ms: HOUR },
  { id: '24h', ms: 24 * HOUR },
  { id: '7d', ms: 7 * 24 * HOUR },
  { id: '30d', ms: 30 * 24 * HOUR },
]
const LIMITS = [200, 500, 1000]

const filter = reactive({ user: '', action: '', text: '', result: '', limit: LIMITS[0] })
const period = ref('all')
const page = ref<AuditPage | null>(null)
const loading = ref(false)
const loadError = ref('')

/** Роль: подпись из roles.*, иначе значение бэкенда как есть */
function roleLabel(role: string): string {
  return te(`roles.${role}`) ? t(`roles.${role}`) : role
}

function formatTime(iso: string): string {
  return formatDateTime(iso) || t('common.dash')
}

// Ответы могут прийти не в порядке отправки (быстрый ввод в фильтр): применяем только последний.
let seq = 0

async function load() {
  const my = ++seq
  loading.value = true
  loadError.value = ''
  const ms = PERIODS.find(p => p.id === period.value)?.ms ?? 0
  const request: AuditFilter = {
    ...filter,
    since: ms ? new Date(Date.now() - ms).toISOString() : '',
    until: '',
  }
  try {
    const result = (await callApp('GetAuditLog', request)) as AuditPage
    if (my !== seq) return
    page.value = { ...result, events: result.events ?? [] }
  } catch (e) {
    if (my !== seq) return
    loadError.value = errorText(e)
  } finally {
    if (my === seq) loading.value = false
  }
}

// Текстовые поля — с задержкой, чтобы не дёргать бэкенд на каждую букву; списки — сразу.
let timer: ReturnType<typeof setTimeout> | undefined
watch(
  () => [filter.user, filter.action, filter.text],
  () => {
    clearTimeout(timer)
    timer = setTimeout(load, 300)
  },
)
watch(() => [filter.result, filter.limit, period.value], load)

onMounted(load)
onBeforeUnmount(() => clearTimeout(timer))
</script>

<style scoped>
.audit-view { padding: 24px; text-align: left; max-width: 1300px; }
.audit-head { display: flex; align-items: flex-start; justify-content: space-between; gap: 16px; margin-bottom: 20px; }
.audit-head h2 { margin: 0; font-size: 22px; color: #89b4fa; }
.audit-sub { margin: 4px 0 0; color: #a6adc8; font-size: 13px; }
.btn:disabled { opacity: 0.5; cursor: not-allowed; }

.audit-warn {
  margin: 0 0 16px; padding: 10px 12px; border-radius: 8px; font-size: 13px; line-height: 1.4;
  background: rgba(249, 226, 175, 0.12); border: 1px solid rgba(249, 226, 175, 0.35); color: #f9e2af;
}

.audit-filters { display: grid; grid-template-columns: repeat(auto-fit, minmax(170px, 1fr)); gap: 10px; margin-bottom: 18px; }
.audit-filters .form-input { width: 100%; box-sizing: border-box; padding: 8px 10px; }

.audit-empty, .audit-more { color: #6c7086; font-size: 13px; margin: 16px 0; }

.audit-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.audit-table th { text-align: left; padding: 8px 12px; color: #a6adc8; font-weight: 600; border-bottom: 1px solid #313244; }
.audit-table td { padding: 9px 12px; border-bottom: 1px solid #232334; vertical-align: top; }
.audit-table tr.denied td { background: rgba(243, 139, 168, 0.05); }
.nowrap { white-space: nowrap; }
.muted { color: #6c7086; }
.detail { color: #a6adc8; font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; word-break: break-word; }

.role-tag { margin-left: 8px; padding: 1px 6px; border-radius: 4px; font-size: 11px; background: #313244; color: #a6adc8; }

.badge { display: inline-block; padding: 2px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; white-space: nowrap; }
.badge-ok { background: rgba(166, 227, 161, 0.15); color: #a6e3a1; }
.badge-failed { background: rgba(249, 226, 175, 0.15); color: #f9e2af; }
.badge-denied { background: rgba(243, 139, 168, 0.15); color: #f38ba8; }
</style>
