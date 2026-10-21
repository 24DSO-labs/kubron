<template>
  <NodeResourcesTable title="Capacity" :fields="resource?.capacity ?? []" />
</template>

<script lang="ts" setup>
import NodeResourcesTable from './NodeResourcesTable.vue'

defineProps<{ resource: any }>()
</script>
