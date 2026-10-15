<template>
  <div class="rd-block">
    <div class="rd-block-head" @click="open = !open">
      <span>{{ title }}</span>
      <span class="rd-arrow" :class="{ open }">›</span>
    </div>
    <div v-if="open" class="rd-block-body"><slot /></div>
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'

withDefaults(defineProps<{ title?: string }>(), { title: 'Metrics' })
const open = ref(false)
</script>
