<template>
  <MetricsBlock v-if="resource.selector">
    <DaemonSetMetricsViewer
      :cluster-name="cluster || ''"
      :namespace="resource.namespace"
      :daemonset-name="resource.name"
      :selector="resource.selector"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import DaemonSetMetricsViewer from '../../metrics/DaemonSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)
</script>
