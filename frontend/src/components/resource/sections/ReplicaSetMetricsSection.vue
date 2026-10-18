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
import { podNameRegex } from '../../../resources/utils'

const props = defineProps<{ resource: any }>()
const cluster = inject(clusterNameKey, null)
const podRegex = computed(() => podNameRegex('replicaset', String(props.resource?.name ?? '')))
</script>