<template>
  <MetricsBlock v-if="clusterName && resource.status === 'Bound' && resource.claimName">
    <PvcMetricsViewer
      :cluster-name="clusterName"
      :namespace="resource.claimNamespace"
      :claim="resource.claimName"
    />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { inject, ref } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import PvcMetricsViewer from '../../metrics/PvcMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

defineProps<{ resource: any }>()
const clusterName = inject(clusterNameKey, ref(undefined))
</script>