<template>
  <NodeResourcesTable title="Allocatable" :fields="resource?.allocatable ?? []" />
</template>

<script lang="ts" setup>
import NodeResourcesTable from './NodeResourcesTable.vue'

defineProps<{ resource: any }>()
</script>
