<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <span class="rv" :aria-label="value">
    <span v-for="(ch, i) in chars" :key="chars.length - i" class="rv-cell">
      <Transition name="rv">
        <span :key="ch" class="rv-ch">{{ ch }}</span>
      </Transition>
    </span>
  </span>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
const props = defineProps<{ value: string }>()
const chars = computed(() => Array.from(props.value))
</script>

<style scoped>
.rv { display: inline-flex; }
.rv-cell {
  position: relative; display: inline-block;
  width: 1ch; height: 1.35em; line-height: 1.35em; overflow: hidden;
}
.rv-ch { display: block; text-align: center; }
.rv-enter-active, .rv-leave-active {
  transition: transform .35s cubic-bezier(.2, .8, .2, 1), opacity .35s;
}
.rv-leave-active { position: absolute; left: 0; right: 0; top: 0; }
.rv-enter-from { transform: translateY(100%); opacity: 0; }
.rv-leave-to   { transform: translateY(-100%); opacity: 0; }
</style>