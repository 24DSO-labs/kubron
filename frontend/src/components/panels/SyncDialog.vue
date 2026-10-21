<template>
  <div class="sd-overlay" @click.self="close">
    <div class="sd" role="dialog" aria-modal="true">
      <div class="sd-head">
        <h3 v-if="!bulk">Sync <span class="sd-name">{{ apps[0].namespace }}/{{ apps[0].name }}</span></h3>
        <h3 v-else>Sync applications <span class="sd-muted">({{ selected.size }} selected)</span></h3>
      </div>

      <div class="sd-body">
        <!-- Массовый режим: выбор приложений, OutOfSync предвыбраны -->
        <template v-if="bulk">
          <input v-model="filter" class="sd-input" type="search" placeholder="Filter applications" />
          <div class="sd-list">
            <label v-for="a in visibleApps" :key="keyOf(a)" class="sd-app" :class="{ off: a.operationOn }">
              <input type="checkbox" :checked="selected.has(keyOf(a))" :disabled="a.operationOn" @change="toggle(a)" />
              <span class="sd-app-name">{{ a.name }}</span>
              <span class="sd-muted">{{ a.namespace }}</span>
              <span class="sd-tag" :class="a.syncStatus === 'OutOfSync' ? 'warn' : 'ok'">{{ a.operationOn ? 'Syncing' : a.syncStatus }}</span>
            </label>
          </div>
        </template>

        <label v-else class="sd-field">
          <span>Revision</span>
          <input v-model="revision" class="sd-input" placeholder="Empty: target revision from the Application spec" />
        </label>

        <label class="sd-check"><input v-model="prune" type="checkbox" /> Prune — delete resources removed from Git</label>
        <label class="sd-check"><input v-model="force" type="checkbox" /> Force — replace resources (may cause downtime)</label>
        <label class="sd-check"><input v-model="dryRun" type="checkbox" /> Dry run — only show what would change</label>

        <div v-if="prune && !dryRun" class="sd-risk" :class="{ danger: pruneTargets.length }">
          <template v-if="pruneTargets.length">
            <strong>{{ pruneTargets.length }} resource(s) will be deleted:</strong>
            <ul>
              <li v-for="t in pruneTargets.slice(0, 8)" :key="t">{{ t }}</li>
            </ul>
            <span v-if="pruneTargets.length > 8">…and {{ pruneTargets.length - 8 }} more</span>
          </template>
          <template v-else>No resources are currently marked for pruning.</template>
        </div>
        <div v-if="force && !dryRun" class="sd-risk danger">Force uses replace/delete-recreate semantics for changed resources.</div>
      </div>

      <div class="sd-foot">
        <button class="sd-btn" :disabled="busy" @click="close">Cancel</button>
        <button v-if="bulk" class="sd-btn" :disabled="busy || !selected.size" @click="refreshSelected">Refresh</button>
        <button class="sd-btn primary" :disabled="busy || !selected.size" @click="submit">
          {{ busy ? 'Starting…' : dryRun ? 'Dry run' : `Sync${bulk ? ` (${selected.size})` : ''}` }}
        </button>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref } from 'vue'
import { refreshArgoApps, runArgoSync } from '../../resources/argoActions'
import type { ArgoApplicationInfo } from '../../types/argocd'

const props = defineProps<{ apps: ArgoApplicationInfo[] }>()
const emit = defineEmits<{ (e: 'close'): void }>()

const bulk = computed(() => props.apps.length > 1)
const keyOf = (a: ArgoApplicationInfo) => `${a.namespace}/${a.name}`

// OutOfSync первыми
const sorted = computed(() =>
  [...props.apps].sort(
    (a, b) =>
      Number(b.syncStatus === 'OutOfSync') - Number(a.syncStatus === 'OutOfSync') ||
      a.name.localeCompare(b.name),
  ),
)

const filter = ref('')
const visibleApps = computed(() => {
  const q = filter.value.trim().toLowerCase()
  return q ? sorted.value.filter(a => `${a.name} ${a.namespace} ${a.project}`.toLowerCase().includes(q)) : sorted.value
})

// Предвыбор: одно приложение — оно само; массово — OutOfSync без идущей операции
const selected = ref(
  new Set(
    props.apps
      .filter(a => (bulk.value ? a.syncStatus === 'OutOfSync' && !a.operationOn : true))
      .map(keyOf),
  ),
)
function toggle(a: ArgoApplicationInfo) {
  const s = new Set(selected.value)
  if (!s.delete(keyOf(a))) s.add(keyOf(a))
  selected.value = s
}
const selectedApps = computed(() => props.apps.filter(a => selected.value.has(keyOf(a))))

const prune = ref(false)
const force = ref(false)
const dryRun = ref(false)
const revision = ref('')
const busy = ref(false)

/** Ресурсы, которые Argo пометил requiresPruning: именно их удалит sync с prune */
const pruneTargets = computed(() =>
  selectedApps.value.flatMap(a =>
    (a.resources ?? []).filter(r => r.requiresPruning && !r.hook).map(r => `${r.kind}/${r.name} (${a.name})`),
  ),
)

const close = () => { if (!busy.value) emit('close') }

async function submit() {
  busy.value = true
  await runArgoSync(selectedApps.value, {
    revision: bulk.value ? '' : revision.value.trim(),
    prune: prune.value,
    force: force.value,
    dryRun: dryRun.value,
  })
  busy.value = false
  emit('close')
}

async function refreshSelected() {
  busy.value = true
  await refreshArgoApps(selectedApps.value, false)
  busy.value = false
  emit('close')
}
</script>

<style scoped>
.sd-overlay { position: fixed; inset: 0; background: rgba(0, 0, 0, 0.6); display: flex; align-items: center; justify-content: center; z-index: 2000; }
.sd { width: 90%; max-width: 560px; max-height: 90vh; display: flex; flex-direction: column; background: #1e1f35; border: 1px solid #3b3d5c; border-radius: 8px; box-shadow: 0 10px 40px rgba(0, 0, 0, 0.5); text-align: left; }
.sd-head { padding: 16px; background: #0f1017; border-bottom: 1px solid #3b3d5c; border-radius: 8px 8px 0 0; }
.sd-head h3 { margin: 0; font-size: 1rem; color: #fff; }
.sd-name { color: #60a5fa; font-family: 'JetBrains Mono', 'Fira Code', monospace; }
.sd-muted { color: #868d8a; font-size: 0.8rem; font-weight: 400; }
.sd-body { padding: 16px 20px; display: flex; flex-direction: column; gap: 12px; overflow-y: auto; }
.sd-input { width: 100%; box-sizing: border-box; padding: 7px 10px; background: #1a1b2e; color: #d4d4d4; border: 1px solid #3b3d5c; border-radius: 4px; font: inherit; font-size: 0.85rem; }
.sd-input:focus { outline: none; border-color: #3b82f6; }
.sd-field { display: flex; flex-direction: column; gap: 6px; font-size: 0.85rem; color: #9ca0b8; }
.sd-list { max-height: 240px; overflow-y: auto; border: 1px solid #2a2b40; border-radius: 4px; }
.sd-app { display: grid; grid-template-columns: auto 1fr auto auto; gap: 10px; align-items: center; padding: 6px 10px; font-size: 0.85rem; color: #d4d4d4; cursor: pointer; border-bottom: 1px solid rgba(255, 255, 255, 0.05); }
.sd-app:hover { background: rgba(255, 255, 255, 0.03); }
.sd-app.off { opacity: 0.5; cursor: not-allowed; }
.sd-app-name { overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sd-tag { padding: 1px 8px; border-radius: 999px; font-size: 0.7rem; font-weight: 600; }
.sd-tag.ok { color: #4ade80; background: rgba(74, 222, 128, 0.12); }
.sd-tag.warn { color: #facc15; background: rgba(250, 204, 21, 0.12); }
.sd-check { display: flex; align-items: center; gap: 8px; font-size: 0.85rem; color: #d4d4d4; cursor: pointer; }
.sd-risk { padding: 10px 12px; border-radius: 4px; font-size: 0.8rem; color: #9ca0b8; background: rgba(255, 255, 255, 0.03); border: 1px solid #2a2b40; }
.sd-risk.danger { color: #fca5a5; background: rgba(239, 68, 68, 0.1); border-color: rgba(239, 68, 68, 0.4); }
.sd-risk ul { margin: 6px 0; padding-left: 18px; font-family: 'Courier New', monospace; }
.sd-foot { display: flex; justify-content: flex-end; gap: 10px; padding: 12px 16px; background: #0f1017; border-top: 1px solid #3b3d5c; border-radius: 0 0 8px 8px; }
.sd-btn { padding: 8px 16px; border-radius: 4px; border: 1px solid #3b3d5c; background: transparent; color: #d4d4d4; font-size: 0.9rem; font-weight: 600; cursor: pointer; }
.sd-btn:hover:not(:disabled) { background: rgba(255, 255, 255, 0.06); }
.sd-btn.primary { background: #3b82f6; border-color: #3b82f6; color: #fff; }
.sd-btn.primary:hover:not(:disabled) { background: #2563eb; }
.sd-btn:disabled { opacity: 0.5; cursor: not-allowed; }
</style>