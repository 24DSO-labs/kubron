<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="hc-view">
    <header class="hc-head">
      <div>
        <h2>Каталог чартов</h2>
        <p class="hc-sub">Поиск по индексам добавленных репозиториев. Индексы обновляются в разделе Repositories.</p>
      </div>
    </header>

    <form class="hc-filters" @submit.prevent="search">
      <input v-model="query" class="form-input" type="search" placeholder="Поиск: nginx, postgres, ingress…" aria-label="Поиск" />
      <select v-model="repoFilter" class="form-input" aria-label="Репозиторий" @change="search">
        <option value="">Все репозитории</option>
        <option v-for="r in repoNames" :key="r" :value="r">{{ r }}</option>
      </select>
    </form>

    <p v-if="error" class="form-error">{{ error }}</p>
    <p v-if="loaded && !repoNames.length" class="hc-empty">Нет репозиториев. Добавьте их в разделе Helm → Repositories.</p>
    <p v-else-if="loaded && !items.length && !error" class="hc-empty">Ничего не найдено.</p>

    <div v-if="items.length" class="hc-layout">
      <div class="hc-list">
        <table class="hc-table">
          <thead><tr><th>Чарт</th><th>Репозиторий</th><th>Версия</th><th>App</th><th>Описание</th></tr></thead>
          <tbody>
            <tr v-for="c in items" :key="c.repo + '/' + c.name" :class="{ sel: isSelected(c) }" tabindex="0"
              @click="select(c)" @keydown.enter="select(c)">
              <td><strong>{{ c.name }}</strong><span v-if="c.deprecated" class="hc-dep">deprecated</span></td>
              <td>{{ c.repo }}</td>
              <td class="hc-mono">{{ c.version }}</td>
              <td class="hc-mono">{{ c.appVersion || '—' }}</td>
              <td class="hc-desc" :title="c.description">{{ c.description }}</td>
            </tr>
          </tbody>
        </table>
        <p v-if="total > items.length" class="hc-empty">Показано {{ items.length }} из {{ total }}: уточните запрос.</p>
      </div>

      <section v-if="selected" class="hc-detail">
        <div class="hc-detail-bar">
          <h3>{{ selected.repo }}/{{ selected.name }}</h3>
          <label class="hc-field">
            Версия
            <select v-model="version" class="form-input" aria-label="Версия">
              <option v-for="v in versions" :key="v.version" :value="v.version">
                {{ v.version }}{{ v.appVersion ? ` (app ${v.appVersion})` : '' }}{{ v.deprecated ? ' · deprecated' : '' }}
              </option>
            </select>
          </label>
          <div class="hc-seg" role="group" aria-label="Раздел">
            <button v-for="t in TABS" :key="t.id" type="button" :class="{ on: tab === t.id }" @click="tab = t.id">{{ t.label }}</button>
          </div>
        </div>

        <p v-if="pkgError" class="form-error">{{ pkgError }}</p>
        <p v-else-if="pkgLoading && !pkg" class="hc-empty">Загрузка чарта…</p>
        <template v-else-if="pkg">
          <!-- README показываем как текст в редакторе: Markdown из внешнего репозитория не рендерим как HTML -->
          <MonacoReadonly v-if="tab === 'readme'" :value="pkg.readme || '# У чарта нет README'" language="markdown" />
          <MonacoReadonly v-else-if="tab === 'values'" :value="pkg.values || '# У чарта нет values.yaml'" language="yaml" />
          <dl v-else class="hc-info">
            <dt>Описание</dt><dd>{{ pkg.description || '—' }}</dd>
            <dt>Версия чарта</dt><dd class="hc-mono">{{ pkg.version }}</dd>
            <dt>Версия приложения</dt><dd class="hc-mono">{{ pkg.appVersion || '—' }}</dd>
            <dt>Сайт</dt><dd class="hc-mono">{{ pkg.home || '—' }}</dd>
            <dt>Исходники</dt><dd class="hc-mono">{{ (pkg.sources || []).join('\n') || '—' }}</dd>
            <dt>Ключевые слова</dt><dd>{{ (pkg.keywords || []).join(', ') || '—' }}</dd>
            <dt>Мейнтейнеры</dt><dd>{{ (pkg.maintainers || []).join(', ') || '—' }}</dd>
          </dl>
        </template>
      </section>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, onBeforeUnmount, onMounted, ref, watch } from 'vue'
import MonacoReadonly from '../resource/MonacoReadonly.vue'
import { callApp } from '../../resources/wails'

interface Chart { repo: string; name: string; version: string; appVersion: string; description: string; deprecated: boolean; versions: number }
interface Ver { version: string; appVersion: string; deprecated: boolean }
interface Pkg {
  repo: string; name: string; version: string; appVersion: string; description: string; home: string
  sources: string[] | null; maintainers: string[] | null; keywords: string[] | null; readme: string; values: string
}

const TABS = [
  { id: 'readme', label: 'README' },
  { id: 'values', label: 'Values' },
  { id: 'info', label: 'Info' },
] as const

const query = ref('')
const repoFilter = ref('')
const repoNames = ref<string[]>([])
const items = ref<Chart[]>([])
const total = ref(0)
const loaded = ref(false)
const error = ref('')

const selected = ref<Chart | null>(null)
const versions = ref<Ver[]>([])
const version = ref('')
const tab = ref<'readme' | 'values' | 'info'>('readme')
const pkg = ref<Pkg | null>(null)
const pkgLoading = ref(false)
const pkgError = ref('')

const isSelected = (c: Chart) => selected.value?.repo === c.repo && selected.value?.name === c.name

let searchSeq = 0
async function search() {
  const my = ++searchSeq
  error.value = ''
  try {
    const res = (await callApp('SearchHelmCharts', query.value, repoFilter.value)) as { items: Chart[]; total: number }
    if (my !== searchSeq) return
    items.value = res.items ?? []
    total.value = res.total ?? 0
    if (selected.value && !items.value.some(isSelected)) selected.value = null
  } catch (e: any) {
    if (my === searchSeq) error.value = e?.message || String(e)
  } finally {
    if (my === searchSeq) loaded.value = true
  }
}

let debounce: ReturnType<typeof setTimeout> | undefined
watch(query, () => {
  clearTimeout(debounce)
  debounce = setTimeout(search, 300)
})
onBeforeUnmount(() => clearTimeout(debounce))

let verSeq = 0
async function select(c: Chart) {
  version.value = '' // иначе watch сначала запросит новый чарт со старой версией
  selected.value = c
  const my = ++verSeq
  versions.value = []
  pkg.value = null
  pkgError.value = ''
  try {
    const v = ((await callApp('GetHelmChartVersions', c.repo, c.name)) as Ver[]) ?? []
    if (my !== verSeq) return
    versions.value = v
    version.value = v[0]?.version ?? c.version // версии отсортированы по убыванию
  } catch (e: any) {
    if (my === verSeq) pkgError.value = e?.message || String(e)
  }
}

let pkgSeq = 0
watch([selected, version], async () => {
  const c = selected.value
  if (!c || !version.value) return
  const my = ++pkgSeq
  pkgLoading.value = true
  pkgError.value = ''
  try {
    const p = (await callApp('GetHelmChartPackage', c.repo, c.name, version.value)) as Pkg
    if (my === pkgSeq) pkg.value = p
  } catch (e: any) {
    if (my === pkgSeq) { pkg.value = null; pkgError.value = e?.message || String(e) }
  } finally {
    if (my === pkgSeq) pkgLoading.value = false
  }
})

onMounted(async () => {
  try {
    repoNames.value = ((await callApp('ListHelmRepos')) as { name: string }[]).map(r => r.name)
  } catch (e: any) {
    error.value = e?.message || String(e)
  }
  await search()
})
</script>

<style scoped>
.hc-view { padding: 24px; text-align: left; }
.hc-head h2 { margin: 0; font-size: 22px; color: #89b4fa; }
.hc-sub { margin: 4px 0 16px; color: #a6adc8; font-size: 13px; }
.hc-filters { display: grid; grid-template-columns: 2fr 1fr; gap: 10px; max-width: 700px; margin-bottom: 16px; }
.hc-filters .form-input { width: 100%; box-sizing: border-box; padding: 8px 10px; }
.hc-empty { color: #6c7086; font-size: 13px; margin: 12px 0; }
.hc-layout { display: grid; grid-template-columns: minmax(380px, 1fr) minmax(420px, 1.3fr); gap: 20px; align-items: start; }
.hc-list { max-height: 75vh; overflow: auto; }
.hc-table { width: 100%; border-collapse: collapse; font-size: 13px; }
.hc-table th { position: sticky; top: 0; background: #1e1e2e; text-align: left; padding: 8px 10px; color: #a6adc8; border-bottom: 1px solid #313244; }
.hc-table td { padding: 8px 10px; border-bottom: 1px solid #232334; }
.hc-table tbody tr { cursor: pointer; }
.hc-table tbody tr:hover, .hc-table tbody tr.sel { background: rgba(137, 180, 250, 0.08); }
.hc-desc { max-width: 280px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; color: #a6adc8; }
.hc-mono { font-family: ui-monospace, SFMono-Regular, Menlo, Consolas, monospace; font-size: 12px; }
.hc-dep { margin-left: 8px; padding: 1px 6px; border-radius: 4px; font-size: 11px; background: rgba(249, 226, 175, 0.15); color: #f9e2af; }
.hc-detail-bar { display: flex; align-items: center; gap: 14px; flex-wrap: wrap; margin-bottom: 10px; }
.hc-detail-bar h3 { margin: 0; font-size: 16px; color: #cdd6f4; }
.hc-field { display: flex; align-items: center; gap: 8px; font-size: 13px; color: #a6adc8; }
.hc-seg { display: inline-flex; border: 1px solid #3b3d5c; border-radius: 4px; overflow: hidden; margin-left: auto; }
.hc-seg button { padding: 4px 12px; background: transparent; color: #9ca3af; border: 0; font: inherit; cursor: pointer; }
.hc-seg button.on { background: #3b82f6; color: #fff; }
.hc-info { display: grid; grid-template-columns: 170px 1fr; gap: 8px 14px; font-size: 13px; margin: 0; }
.hc-info dt { color: #6c7086; }
.hc-info dd { margin: 0; white-space: pre-line; word-break: break-word; }
</style>
