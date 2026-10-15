<template>
  <MetricsBlock v-if="resource.selector">
    <DeploymentMetricsViewer
      :cluster-name="cluster || ''"
      :namespace="resource.namespace"
      :deployment-name="resource.name"
      :selector="resource.selector"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import DeploymentMetricsViewer from '../../metrics/DeploymentMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)
</script>
