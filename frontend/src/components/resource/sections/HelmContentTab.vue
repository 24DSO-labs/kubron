<template>
  <div class="hc">
    <div class="hc-bar">
      <label v-if="revisions.length > 1" class="hc-field">
        Revision
        <select v-model.number="revision" class="hc-select" aria-label="Revision">
          <option v-for="r in revisions" :key="r.revision" :value="r.revision">#{{ r.revision }} · {{ r.status }}</option>
        </select>
      </label>
      <div v-if="kind === 'values'" class="hc-seg" role="group" aria-label="Values mode">
        <button :class="{ on: mode === 'user' }" @click="mode = 'user'">User-supplied</button>
        <button :class="{ on: mode === 'all' }" @click="mode = 'all'">All (with defaults)</button>
      </div>
      <button class="rd-btn hc-copy" :disabled="!text" @click="copy">{{ copied ? 'Copied' : 'Copy' }}</button>
    </div>

    <div v-if="error" class="rd-empty">{{ error }}</div>
    <div v-else-if="loading && !content" class="rd-empty">Loading…</div>
    <div v-else-if="kind === 'notes' && !text" class="rd-empty">This release has no notes</div>
    <MonacoReadonly v-else :value="text" :language="language" />
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, shallowRef, watch } from 'vue'
import MonacoReadonly from '../MonacoReadonly.vue'
import { loadHelmContent } from '../../../resources/helmContent'
import { itemName, itemNs } from '../../../resources/utils'
import type { k8s } from '../../../../wailsjs/go/models'

const props = defineProps<{ resource: any; kind: 'values' | 'notes' | 'manifest' }>()

const ns = computed(() => itemNs(props.resource))
const name = computed(() => itemName(props.resource))
const revisions = computed<{ revision: number; status: string }[]>(() => props.resource?.history ?? [])

const revision = ref<number>(Number(props.resource?.revision) || 0)
// Релиз обновился (появилась новая ревизия): показываем её
watch(() => props.resource?.revision, r => { if (r) revision.value = Number(r) })

const mode = ref<'user' | 'all'>('user')
const content = shallowRef<k8s.HelmReleaseContent | null>(null)
const loading = ref(false)
const error = ref('')
const copied = ref(false)

let seq = 0
async function load() {
  const my = ++seq
  loading.value = true
  error.value = ''
  try {
    const c = await loadHelmContent(ns.value, name.value, revision.value)
    if (my !== seq) return
    content.value = c
  } catch (e: any) {
    if (my !== seq) return
    content.value = null
    error.value = e?.message || String(e)
  } finally {
    if (my === seq) loading.value = false
  }
}
watch([ns, name, revision], load, { immediate: true })

const text = computed(() => {
  const c = content.value
  if (!c) return ''
  switch (props.kind) {
    case 'values':
      return mode.value === 'all' ? c.computedValues : c.values || '# No user-supplied values: chart defaults are used\n'
    case 'notes':
      return c.notes
    default:
      return c.manifest
  }
})
const language = computed(() => (props.kind === 'notes' ? 'plaintext' : 'yaml'))

async function copy() {
  try {
    await navigator.clipboard.writeText(text.value)
    copied.value = true
    setTimeout(() => { copied.value = false }, 1500)
  } catch { /* буфер обмена недоступен */ }
}
</script>

<style scoped>
.hc-bar { display: flex; align-items: center; gap: 12px; padding: 8px 0; flex-wrap: wrap; font-size: 0.85rem; }
.hc-field { display: flex; align-items: center; gap: 6px; color: #9ca3af; }
.hc-select { padding: 5px 8px; background: #1a1b2e; color: #d4d4d4; border: 1px solid #3b3d5c; border-radius: 4px; font: inherit; }
.hc-seg { display: inline-flex; border: 1px solid #3b3d5c; border-radius: 4px; overflow: hidden; }
.hc-seg button { padding: 4px 10px; background: transparent; color: #9ca3af; border: 0; font: inherit; cursor: pointer; }
.hc-seg button.on { background: #3b82f6; color: #fff; }
.hc-copy { margin-left: auto; }
</style>