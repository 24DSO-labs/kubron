<template>
  <NodeMetricsPanel v-if="clusterName && nodeName" :cluster-name="clusterName" :node-name="nodeName" />
</template>

<script lang="ts" setup>
import { computed, inject, ref } from 'vue'
import NodeMetricsPanel from '../../metrics/NodeMetricsPanel.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()

const clusterName = inject(clusterNameKey, ref(undefined))
const nodeName = computed(() => props.resource?.name || '')
</script>