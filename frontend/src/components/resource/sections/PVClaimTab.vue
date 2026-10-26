<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="claimName" class="pv-tab">
    <div class="rd-row">
      <span class="rd-label">Type</span>
      <span class="rd-value">PersistentVolumeClaim</span>
    </div>
    <div class="rd-row">
      <span class="rd-label">Name</span>
      <span class="rd-value">
        <a class="rd-link" @click="emit('open', { id: 'persistentvolumeclaim', ns: claimNs, name: claimName })">
          {{ claimName }}
        </a>
      </span>
    </div>
    <div class="rd-row">
      <span class="rd-label">Namespace</span>
      <span class="rd-value">{{ claimNs }}</span>
    </div>
  </div>
  <div v-else class="rd-empty">Volume is not bound to a claim</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const claimName = computed<string>(() => props.resource?.claimName ?? '')
const claimNs = computed<string>(() => props.resource?.claimNamespace ?? '')
</script>

<style scoped>
.pv-tab { padding: 4px 0; }
</style>