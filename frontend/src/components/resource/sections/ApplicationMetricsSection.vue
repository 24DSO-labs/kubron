<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <MetricsBlock v-if="cluster && owners.length">
    <PodSetMetricsViewer :cluster-name="cluster" :namespace="namespace" :owners="owners" />
  </MetricsBlock>
</template>

<script lang="ts" setup>
import { computed, inject, ref } from 'vue'
import MetricsBlock from './MetricsBlock.vue'
import PodSetMetricsViewer from '../../metrics/PodSetMetricsViewer.vue'
import { clusterNameKey } from '../../../resources/context'
import { applicationOwners } from '../../../resources/applications'

/** Метрики приложения: суммарно по подам всех его workload-ов */
const props = defineProps<{ resource: any }>()

const cluster = inject(clusterNameKey, ref<string | undefined>(undefined))
const namespace = computed(() => String(props.resource?.namespace ?? ''))
const owners = computed(() => applicationOwners(props.resource))
</script>
