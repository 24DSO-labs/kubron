<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div v-if="clusterName" class="rd-block">
    <div class="rd-block-head" @click="open = !open">
      <span>Metrics</span>
      <span class="rd-arrow" :class="{ open }">›</span>
    </div>
    <div v-if="open" class="rd-block-body">
      <PodMetricsViewer
        :cluster-name="clusterName"
        :namespace="resource.namespace"
        :pod-name="resource.name"
      />
    </div>
  </div>
</template>

<script lang="ts" setup>
import { ref, inject } from 'vue'
import PodMetricsViewer from '../../metrics/PodMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()
const open = ref(false)
const clusterName = inject(clusterNameKey, ref(undefined))
</script>
