<template>
  <MetricsBlock v-if="clusterName && namespace">
    <PodSetMetricsViewer :cluster-name="clusterName" :namespace="namespace" />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { computed, inject, ref } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import PodSetMetricsViewer from '../../metrics/PodSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'

const props = defineProps<{ resource: any }>()
const clusterName = inject(clusterNameKey, ref(undefined))

// Namespace — кластерный ресурс: его собственное имя и есть namespace для запросов
const namespace = computed<string>(() => String(props.resource?.name ?? ''))
</script>