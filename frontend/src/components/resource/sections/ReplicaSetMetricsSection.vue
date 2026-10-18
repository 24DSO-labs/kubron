<template>
  <MetricsBlock v-if="resource.selector">
    <ReplicaSetMetricsViewer
      :cluster-name="cluster || ''"
      :namespace="resource.namespace"
      :replicaset-name="resource.name"
      :selector="resource.selector"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import ReplicaSetMetricsViewer from '../../metrics/ReplicaSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)
</script>
