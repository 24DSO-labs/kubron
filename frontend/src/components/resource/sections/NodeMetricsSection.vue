<template>
  <MetricsBlock v-if="clusterName && nodeName">
    <NodeMetricsViewer :cluster-name="clusterName" :node-name="nodeName" />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { computed, inject, ref } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import NodeMetricsViewer from '../../metrics/NodeMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()

const clusterName = inject(clusterNameKey, ref(undefined))
const nodeName = computed(() => props.resource?.name || '')
</script>
