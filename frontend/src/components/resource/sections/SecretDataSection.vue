<template>
  <div class="rd-row rd-click" @click="open = !open">
    <span class="rd-label">Data<span class="rd-arrow" :class="{ open }">›</span></span>
    <span v-if="!open" class="rd-muted">{{ keys.length }} keys</span>
    <div v-else-if="keys.length" class="rd-wide" @click.stop>
      <div v-for="k in keys" :key="k" class="sec-item">
        <div class="sec-head">
          <span class="sec-key">{{ k }}</span>
          <div class="sec-actions">
            <button class="rd-btn" @click.stop="revealed[k] = !revealed[k]">{{ revealed[k] ? 'Hide' : 'Reveal' }}</button>
            <button class="rd-btn" @click.stop="copy(k)">Copy</button>
          </div>
        </div>
        <pre v-if="revealed[k]" class="sec-value">{{ decode(k) }}</pre>
        <span v-else class="sec-mask">{{ mask(data[k]) }}</span>
      </div>
    </div>
    <span v-else class="rd-muted">—</span>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed, reactive } from 'vue'

const props = defineProps<{ resource: any }>()
const open = ref(false)
const data = computed(() => props.resource?.data ?? {})
const keys = computed(() => Object.keys(data.value))
const revealed = reactive<Record<string, boolean>>({})

function decode(key: string): string {
  const v = data.value[key]
  try { return atob(v) } catch { return v }
}
function mask(v: string): string {
  const len = v?.length ?? 0
  return '•'.repeat(Math.min(len, 20)) + (len > 20 ? ` (${len} chars)` : '')
}
function copy(key: string) {
  navigator.clipboard.writeText(decode(key)).catch(() => {})
}
</script>

<style scoped>
.sec-item { margin-bottom: 10px; padding: 8px; background: rgba(255,255,255,0.02); border-radius: 4px; border: 1px solid #2a2b40; }
.sec-head { display: flex; align-items: center; justify-content: space-between; margin-bottom: 6px; }
.sec-key { font-weight: 600; color: #60a5fa; font-size: 0.85rem; }
.sec-actions { display: flex; gap: 6px; }
.sec-value { margin: 0; padding: 8px; background: #1e1f35; border-radius: 3px; font-size: 0.8rem; color: #9ca0b8; white-space: pre-wrap; word-break: break-word; max-height: 200px; overflow-y: auto; }
.sec-mask { color: #6b6d8a; font-size: 0.8rem; }
</style>
