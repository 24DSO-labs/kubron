<template>
  <div v-if="policies.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead><tr><th>Name</th><th>Policy types</th></tr></thead>
      <tbody>
        <tr
          v-for="p in policies"
          :key="`${p.namespace}/${p.name}`"
          class="rd-click"
          @click="emit('open', { id: 'networkpolicy', ns: p.namespace, name: p.name })"
        >
          <td class="rd-link">{{ p.name }}</td>
          <td>{{ (p.policyTypes ?? []).join(', ') || '—' }}</td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">No network policies select the pods of this service</div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()
const policies = computed<any[]>(() => props.resource?.networkPolicies ?? [])
</script>