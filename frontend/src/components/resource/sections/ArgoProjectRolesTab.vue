<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="roles.length" class="rd-stack">
    <div v-for="r in roles" :key="r.name" class="rd-row">
      <span class="rd-label">{{ r.name }}</span>
      <div>
        <div v-if="r.description" class="rd-muted">{{ r.description }}</div>
        <div v-if="r.groups?.length" class="rd-badges">
          <span v-for="g in r.groups" :key="g" class="rd-badge rd-unk">{{ g }}</span>
        </div>
        <pre v-if="r.policies?.length" class="rd-mono">{{ r.policies.join('\n') }}</pre>
      </div>
    </div>
  </div>
  <div v-else class="rd-empty">No roles</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ArgoProjectInfo } from '../../../types/argocd'

const props = defineProps<{ resource: Partial<ArgoProjectInfo> }>()
const roles = computed(() => props.resource?.roles ?? [])
</script>
