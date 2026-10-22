<template>
  <div class="ad">
    <div v-if="configured === null && !statusError" class="rd-empty">Loading…</div>
    <div v-else-if="statusError" class="rd-empty">{{ statusError }}</div>

    <!-- Токен не задан: объяснение и форма -->
    <div v-else-if="!configured" class="ad-setup">
      <p>
        Diff между git и кластером считает сам Argo CD, поэтому нужен токен его API.
        Создайте его командой <code>argocd account generate-token --account &lt;имя&gt;</code>
        (аккаунту нужно право <code>get</code> на приложения).
      </p>
      <p class="rd-muted">
        Токен хранится только на этом компьютере в файле с правами 0600, не показывается в интерфейсе
        и не попадает в журнал аудита. Значения Secret в diff скрываются.
      </p>
      <form v-if="canManage" class="ad-form" @submit.prevent="saveToken">
        <input
          v-model="tokenInput"
          class="ad-input"
          type="password"
          autocomplete="off"
          spellcheck="false"
          placeholder="Argo CD API token"
          aria-label="Argo CD API token"
        />
        <button class="rd-btn" type="submit" :disabled="saving || !tokenInput.trim()">Save token</button>
      </form>
      <p v-else class="rd-muted">Сохранить токен может пользователь с правом «Управление кластерами».</p>
      <p v-if="error" class="ad-error">{{ error }}</p>
    </div>

    <!-- Токен задан -->
    <template v-else>
      <div class="ad-toolbar">
        <button class="rd-btn" :disabled="loading" @click="load">{{ diff ? 'Reload diff' : 'Load diff' }}</button>
        <label v-if="diff" class="ad-check"><input v-model="onlyChanged" type="checkbox" /> Only differences</label>
        <span v-if="diff" class="rd-muted">{{ diff.changed }} of {{ diff.total }} resources differ</span>
        <button v-if="canManage" class="rd-btn ad-remove" :disabled="loading" @click="removeToken">Remove token</button>
      </div>
      <p v-if="!diff && !loading && !error" class="rd-muted">
        Загрузка откроет туннель к argocd-server (проброс порта) и запросит состояние ресурсов у Argo CD.
      </p>
      <p v-if="loading" class="rd-muted">Loading diff…</p>
      <p v-if="error" class="ad-error">{{ error }}</p>

      <template v-if="diff && !loading">
        <div v-if="diff.truncated" class="ad-note">Показаны не все ресурсы: список обрезан</div>
        <div v-if="!visible.length" class="rd-empty">{{ diff.changed ? 'No resources match' : 'Cluster matches git' }}</div>
        <div v-for="r in visible" :key="keyOf(r)" class="ad-item">
          <button class="ad-head" type="button" :disabled="!r.diff" @click="toggle(keyOf(r))">
            <span class="ad-caret">{{ r.diff ? (open.has(keyOf(r)) ? '▾' : '▸') : '' }}</span>
            <span class="rd-badge" :class="tone(r.state)">{{ label(r.state) }}</span>
            <span class="ad-kind">{{ r.kind }}<span v-if="r.hook" class="rd-muted"> (hook)</span></span>
            <span class="ad-name">{{ r.name }}</span>
            <span class="rd-muted ad-ns">{{ r.namespace }}</span>
          </button>
          <pre v-if="open.has(keyOf(r)) && r.diff" class="ad-diff"><span v-for="(l, i) in lines(r.diff)" :key="i" :class="lineClass(l)">{{ l }}</span></pre>
          <div v-if="open.has(keyOf(r)) && r.truncated" class="ad-note">Diff обрезан из-за размера</div>
        </div>
      </template>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { can } from '../../../composables/useAuth'
import { Perm } from '../../../types/auth'
import type { ArgoApplicationInfo, ArgoAPIStatus, ArgoDiff, ArgoDiffState, ArgoResourceDiff } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoApplicationInfo> }>()

const canManage = computed(() => can(Perm.ClusterManage))

const configured = ref<boolean | null>(null)
const statusError = ref('')
const tokenInput = ref('')
const saving = ref(false)
const loading = ref(false)
const error = ref('')
const diff = ref<ArgoDiff | null>(null)
const onlyChanged = ref(true)
const open = ref<Set<string>>(new Set())

const errText = (e: any) => e?.message || String(e)
// Бэкенд возвращает ErrArgoTokenNotSet текстом: так вкладка возвращается к форме, если токен удалили снаружи
const isTokenMissing = (msg: string) => msg.includes('токен Argo CD API не задан')

async function loadStatus() {
  try {
    const st = (await callApp('GetArgoAPIStatus')) as ArgoAPIStatus
    configured.value = !!st?.configured
    statusError.value = ''
  } catch (e: any) {
    statusError.value = errText(e)
  }
}
onMounted(loadStatus)

async function saveToken() {
  const token = tokenInput.value
  saving.value = true
  error.value = ''
  try {
    await callApp('SetArgoAPIToken', token)
    configured.value = true
  } catch (e: any) {
    error.value = errText(e)
  } finally {
    tokenInput.value = '' // токен не остаётся в состоянии компонента ни при успехе, ни при ошибке
    saving.value = false
  }
}

async function removeToken() {
  if (!window.confirm('Удалить сохранённый токен Argo CD API для этого кластера?')) return
  error.value = ''
  try {
    await callApp('DeleteArgoAPIToken')
    configured.value = false
    diff.value = null
  } catch (e: any) {
    error.value = errText(e)
  }
}

let seq = 0
async function load() {
  const ns = props.resource?.namespace
  const name = props.resource?.name
  if (!ns || !name) return
  const my = ++seq
  loading.value = true
  error.value = ''
  try {
    const d = (await callApp('GetArgoApplicationDiff', ns, name)) as ArgoDiff
    if (my !== seq) return
    diff.value = d
    open.value = new Set()
  } catch (e: any) {
    if (my !== seq) return
    const msg = errText(e)
    if (isTokenMissing(msg)) configured.value = false
    else error.value = msg
  } finally {
    if (my === seq) loading.value = false
  }
}

// Открыто другое приложение: старый diff к нему не относится
watch(() => `${props.resource?.namespace}/${props.resource?.name}`, () => {
  seq++
  diff.value = null
  error.value = ''
  open.value = new Set()
  loading.value = false
})

const visible = computed<ArgoResourceDiff[]>(() =>
  (diff.value?.resources ?? []).filter(r => !onlyChanged.value || r.state !== 'InSync'))

const keyOf = (r: ArgoResourceDiff) => `${r.group}/${r.kind}/${r.namespace}/${r.name}`

function toggle(k: string) {
  const next = new Set(open.value)
  if (!next.delete(k)) next.add(k)
  open.value = next
}

const label = (s: ArgoDiffState) =>
  ({ InSync: 'In sync', Modified: 'Modified', Missing: 'Missing in cluster', Extra: 'Not in git' })[s] ?? s
const tone = (s: ArgoDiffState) =>
  ({ InSync: 'rd-ok', Modified: 'rd-warn', Missing: 'rd-warn', Extra: 'rd-err' })[s] ?? 'rd-unk'

const lines = (d: string) => d.split('\n')
const lineClass = (l: string) =>
  l.startsWith('@@') ? 'ad-l-hunk' : l.startsWith('+') ? 'ad-l-add' : l.startsWith('-') ? 'ad-l-del' : 'ad-l-ctx'
</script>

<style scoped>
.ad-setup p { margin: 0 0 10px; line-height: 1.5; font-size: 0.85rem; }
.ad-setup code { font-family: 'Courier New', monospace; background: #1a1d2e; padding: 1px 5px; border-radius: 3px; }
.ad-form { display: flex; gap: 8px; margin: 12px 0; }
.ad-input {
  flex: 1; min-width: 200px; padding: 6px 10px; background: #1a1b2e; color: #d4d4d4;
  border: 1px solid #3b3d5c; border-radius: 4px; font: inherit;
}
.ad-error { color: #f87171; font-size: 0.85rem; margin: 8px 0; }
.ad-toolbar { display: flex; align-items: center; gap: 12px; padding: 8px 0; flex-wrap: wrap; font-size: 0.85rem; }
.ad-check { display: flex; align-items: center; gap: 6px; color: #d4d4d4; cursor: pointer; }
.ad-remove { margin-left: auto; }
.ad-note { padding: 6px 0; font-size: 0.8rem; color: #facc15; }
.ad-item { border-bottom: 1px solid #22263b; }
.ad-head {
  display: flex; align-items: center; gap: 10px; width: 100%; padding: 8px 4px; background: none; border: 0;
  color: inherit; font: inherit; text-align: left; cursor: pointer;
}
.ad-head:disabled { cursor: default; }
.ad-head:not(:disabled):hover { background: #20243a; }
.ad-caret { width: 12px; color: #8b91ad; }
.ad-kind { color: #8b91ad; font-size: 0.8rem; }
.ad-name { font-family: 'Courier New', monospace; font-size: 0.85rem; }
.ad-ns { margin-left: auto; font-size: 0.8rem; }
.ad-diff {
  margin: 0 0 10px; padding: 8px 10px; overflow: auto; max-height: 420px; background: #12141f;
  border: 1px solid #2a2f47; border-radius: 4px; font-family: 'Courier New', monospace; font-size: 0.8rem; line-height: 1.45;
}
.ad-diff span { display: block; white-space: pre; }
.ad-l-add { color: #4ade80; background: rgba(74, 222, 128, 0.08); }
.ad-l-del { color: #f87171; background: rgba(248, 113, 113, 0.08); }
.ad-l-hunk { color: #60a5fa; }
.ad-l-ctx { color: #9ca3af; }
</style>
