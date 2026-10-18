<template>
  <MetricsBlock v-if="resource.selector">
    <StatefulSetMetricsViewer
      :cluster-name="cluster || ''"
      :namespace="resource.namespace"
      :statefulset-name="resource.name"
      :selector="resource.selector"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import StatefulSetMetricsViewer from '../../metrics/StatefulSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)
</script>