<template>
  <div class="rd-row">
    <span class="rd-label">Used by</span>
    <span v-if="!list.length" class="rd-muted">—</span>
    <span v-else class="rd-value rd-links">
      <a
        v-for="u in list"
        :key="`${u.namespace}/${u.name}`"
        class="rd-link"
        @click="emit('open', { id: 'ingress', ns: u.namespace, name: u.name })"
      >{{ u.namespace }}/{{ u.name }}<span v-if="u.implicit" class="rd-muted"> (default)</span></a>
    </span>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string }): void }>()
const list = computed<any[]>(() => props.resource?.usedBy ?? [])
</script>