<template>
  <div v-if="entries.length" class="rd-wide">
    <div class="rd-label" style="margin: 12px 0 8px">Data</div>
    <div v-for="[key, val] in entries" :key="key" class="rd-data-item">
      <div class="rd-data-key">{{ key }}</div>
      <pre class="rd-data-value">{{ val }}</pre>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const entries = computed(() => Object.entries(props.resource?.data ?? {}))
</script>
