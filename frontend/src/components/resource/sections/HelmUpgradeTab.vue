<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="hu">
    <div v-if="!allowed" class="rd-empty">Изменение values требует права на запись</div>
    <div v-else-if="error && !loaded" class="rd-empty">{{ error }}</div>
    <div v-else-if="!loaded" class="rd-empty">Loading…</div>
    <template v-else>
      <p class="rd-muted hu-hint">
        Меняются только values, версия чарта остаётся прежней. Слева текущие user-values, справа редактируемые.
        Удалённый ключ возвращается к значению по умолчанию из чарта.
      </p>

      <MonacoDiff
        v-show="step === 'edit'"
        :original="original"
        v-model:modified="edited"
        editable
      />
      <template v-if="step === 'preview'">
        <div class="rd-muted hu-hint">Изменения манифеста (значения Secret скрыты)</div>
        <MonacoDiff :original="manifestOld" :modified="manifestNew" />
      </template>

      <p v-if="error" class="hu-error">{{ error }}</p>

      <div class="hu-actions">
        <template v-if="step === 'edit'">
          <button class="rd-btn" :disabled="!dirty || busy" @click="preview">{{ busy ? 'Считаю…' : 'Preview changes' }}</button>
          <button class="rd-btn" :disabled="!dirty || busy" @click="edited = original">Reset</button>
        </template>
        <template v-else>
          <button class="rd-btn" :disabled="busy" @click="step = 'edit'">Back</button>
          <button class="rd-btn hu-apply" :disabled="busy" @click="apply">{{ busy ? 'Применяю…' : 'Apply upgrade' }}</button>
        </template>
      </div>
    </template>
  </div>
</template>

<script lang="ts" setup>
import { computed, ref, watch } from 'vue'
import MonacoDiff from '../MonacoDiff.vue'
import { callApp } from '../../../resources/wails'
import { askConfirm } from '../../../resources/confirm'
import { reloadActive } from '../../../resources/store'
import { itemName, itemNs } from '../../../resources/utils'
import { can } from '../../../composables/useAuth'
import { useNotifications } from '../../../composables/useNotifications'
import { Perm } from '../../../types/auth'

const props = defineProps<{ resource: any }>()
const notifications = useNotifications()

const ns = computed(() => itemNs(props.resource))
const name = computed(() => itemName(props.resource))
const allowed = computed(() => can(Perm.Write) && can(Perm.SecretsRead))

const step = ref<'edit' | 'preview'>('edit')
const loaded = ref(false)
const busy = ref(false)
const error = ref('')
const original = ref('')
const edited = ref('')
const manifestOld = ref('')
const manifestNew = ref('')
const dirty = computed(() => edited.value !== original.value)

let seq = 0
async function load() {
  const my = ++seq
  loaded.value = false
  error.value = ''
  step.value = 'edit'
  if (!allowed.value) return
  try {
    // revision 0 — последняя; в кеш такой запрос не попадает
    const c = await callApp('GetHelmReleaseContent', ns.value, name.value, 0)
    if (my !== seq) return
    original.value = c.values ?? ''
    edited.value = original.value
    loaded.value = true
  } catch (e: any) {
    if (my === seq) error.value = e?.message || String(e)
  }
}
// Релиз сменился или появилась новая ревизия: перечитываем исходные values
watch(() => [ns.value, name.value, props.resource?.revision], load, { immediate: true })

async function preview() {
  busy.value = true
  error.value = ''
  try {
    const p = await callApp('PreviewHelmUpgrade', ns.value, name.value, edited.value)
    manifestOld.value = p.currentManifest
    manifestNew.value = p.newManifest
    step.value = 'preview'
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    busy.value = false
  }
}

async function apply() {
  if (!(await askConfirm(`Upgrade ${name.value} with the edited values? A new revision will be created.`))) return
  busy.value = true
  error.value = ''
  try {
    const rev = await callApp('UpgradeHelmRelease', ns.value, name.value, edited.value)
    notifications.notifySuccess(`${name.value}: upgraded, revision #${rev}`)
    await reloadActive()
    await load()
  } catch (e: any) {
    error.value = e?.message || String(e)
  } finally {
    busy.value = false
  }
}
</script>

<style scoped>
.hu-hint { font-size: 0.85rem; margin: 0 0 8px; }
.hu-actions { display: flex; gap: 8px; margin-top: 10px; }
.hu-actions .rd-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.hu-error { color: #f87171; font-size: 0.85rem; margin: 8px 0 0; white-space: pre-wrap; }
</style>
