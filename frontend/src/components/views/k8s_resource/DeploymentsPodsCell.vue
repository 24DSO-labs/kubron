<template>
  <span :class="item.readyReplicas === item.replicas ? 'text-green' : 'text-yellow'">
    {{ item.readyReplicas }}/{{ item.replicas }}
  </span>
</template>

<script lang="ts" setup>
defineProps<{
  item: any
  col: any
}>()
</script>

<style scoped>
.text-green {
  color: #4ade80;
}

.text-yellow {
  color: #facc15;
}
</style>
