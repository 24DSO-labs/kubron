<template>
  <div v-if="conditions.length" class="rd-row">
    <span class="rd-label">Conditions</span>
    <div class="rd-badges">
      <span
        v-for="c in conditions"
        :key="c.type"
        class="rd-badge"
        :class="badgeClass(c.status)"
        :title="c.message || c.reason || ''"
      >{{ c.type }}</span>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { badgeClass } from '../../../resources/status'

const props = defineProps<{ resource: any }>()
const conditions = computed<any[]>(() =>
  Array.isArray(props.resource?.conditions) ? props.resource.conditions : [],
)
</script>
