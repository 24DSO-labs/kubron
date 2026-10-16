<template>
  <div v-if="taints.length" class="rd-row">
    <span class="rd-label">Taints</span>
    <div class="rd-badges">
      <span v-for="(t, i) in taints" :key="i" class="rd-badge rd-unk" :title="t.effect">
        {{ taintText(t) }}
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()

const taints = computed<any[]>(() =>
  Array.isArray(props.resource?.taints) ? props.resource.taints : [],
)

function taintText(t: any): string {
  const kv = t.value ? `${t.key}=${t.value}` : t.key
  return `${kv}:${t.effect}`
}
</script>