<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <MetricsBlock v-if="clusterName && resource.name && resource.namespace">
    <IngressMetricsViewer
      :cluster-name="clusterName"
      :namespace="resource.namespace"
      :ingress="resource.name"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject, ref } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import IngressMetricsViewer from '../../metrics/IngressMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const clusterName = inject(clusterNameKey, ref(undefined))
</script>