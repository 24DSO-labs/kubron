<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="rd-block">
    <div class="rd-block-head" @click="open = !open">
      <span>{{ title || $t('defs.columns.metrics') }}</span>
      <span class="rd-arrow" :class="{ open }">›</span>
    </div>
    <div v-if="open" class="rd-block-body"><slot /></div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

defineProps<{ title?: string }>()
const open = ref(false)
</script>
