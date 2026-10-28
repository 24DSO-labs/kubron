<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <NodeResourcesTable :title="$t('defs.tabs.capacity')" :fields="resource?.capacity ?? []" />
</template>

<script lang="ts" setup>
import NodeResourcesTable from './NodeResourcesTable.vue'

defineProps<{ resource: any }>()
</script>
