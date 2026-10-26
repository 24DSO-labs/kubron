<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="entries.length" class="rd-row rd-click" @click="open = !open">
    <span class="rd-label">{{ title }}<span class="rd-arrow" :class="{ open }">›</span></span>
    <span v-if="!open" class="rd-muted">{{ entries.length }} {{ title }}</span>
    <div v-else class="rd-pairs">
      <span v-for="[k, v] in entries" :key="k">
        <span class="rd-key">{{ k }}</span>=<span class="rd-val">{{ v }}</span>
      </span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, computed } from 'vue'

const props = defineProps<{ title: string; data?: Record<string, any> | null }>()
const open = ref(false)
const entries = computed(() => Object.entries(props.data ?? {}))
</script>
