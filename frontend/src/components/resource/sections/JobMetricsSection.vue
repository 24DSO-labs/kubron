<template>
  <MetricsBlock v-if="cluster">
    <PodSetMetricsViewer
      :cluster-name="cluster || ''"
      :namespace="resource.namespace"
      :pod-regex="podRegex"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { computed, inject } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import PodSetMetricsViewer from '../../metrics/PodSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)

const podRegex = computed(() => {
  const name = String(props.resource?.name ?? '').replace(/\./g, '[.]')
  return `${name}-([0-9]+-)?[a-z0-9]{5}`
})
</script>