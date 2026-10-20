<template>
  <div class="rd-wide">
    <div class="es-title">Endpoints</div>
    <div v-if="entries.length" class="rd-table-wrap">
      <table class="rd-table">
        <thead><tr><th>Address</th><th>Node</th><th>Zone</th><th>Target</th><th>Ready</th><th>Serving</th><th>Terminating</th></tr></thead>
        <tbody>
          <tr v-for="(e, i) in entries" :key="i">
            <td class="rd-mono">{{ e.address }}</td>
            <td>
              <a v-if="e.node" class="rd-link" @click="emit('open', { id: 'node', ns: '', name: e.node })">{{ e.node }}</a>
              <template v-else>—</template>
            </td>
            <td>{{ e.zone || '—' }}</td>
            <td>
              <a v-if="e.target" class="rd-link"
                 @click="emit('open', { id: String(e.target.kind).toLowerCase(), ns: e.target.namespace || resource.namespace, name: e.target.name })">
                {{ e.target.kind }}/{{ e.target.name }}
              </a>
              <template v-else>—</template>
            </td>
            <td><span class="rd-badge" :class="e.ready ? 'rd-ok' : 'rd-warn'">{{ e.ready }}</span></td>
            <td><span class="rd-badge" :class="e.serving ? 'rd-ok' : 'rd-warn'">{{ e.serving }}</span></td>
            <td><span class="rd-badge" :class="e.terminating ? 'rd-warn' : 'rd-unk'">{{ e.terminating }}</span></td>
          </tr>
        </tbody>
      </table>
    </div>
    <div v-else class="rd-muted">No endpoints</div>
  </div>
</template>

<script lang="ts" setup>
import { computed } from 'vue'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{ (e: 'open', p: { id: string; ns: string; name: string }): void }>()
const entries = computed<any[]>(() => props.resource?.entries ?? [])
</script>

<style scoped>
.es-title { font-weight: 700; color: #fff; padding: 8px 0; text-align: left; }
</style>