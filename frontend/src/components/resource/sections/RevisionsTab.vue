<template>
  <div v-if="sorted.length" class="rd-table-wrap">
    <table class="rd-table">
      <thead>
        <tr><th>Revision</th><th>Name</th><th>Pods</th><th>Age</th><th>Images</th><th></th></tr>
      </thead>
      <tbody>
        <tr
          v-for="r in sorted"
          :key="r.name"
          class="rd-click"
          @click="emit('open', { id: 'replicaset', ns: namespace, name: r.name })"
        >
          <td class="rd-link">{{ r.revision }}</td>
          <td>{{ r.name }}</td>
          <td class="rd-mono">{{ r.ready }}/{{ r.replicas }}</td>
          <td class="rd-muted">{{ r.age || '—' }}</td>
          <td class="rd-mono">{{ r.images || '—' }}</td>
          <td @click.stop>
            <button class="rd-btn" @click="rollback(r.revision)">Rollback</button>
          </td>
        </tr>
      </tbody>
    </table>
  </div>
  <div v-else class="rd-empty">{{ loading ? '⏳ Загрузка...' : 'No revisions available' }}</div>
</template>

<script lang="ts" setup>
import { ref, computed, watch } from 'vue'
import { callApp } from '../../../resources/wails'
import { reloadActive } from '../../../resources/store'
import { askConfirm } from '../../../resources/confirm'
import { useNotifications } from '../../../composables/useNotifications'
import { itemName, itemNs } from '../../../resources/utils'

const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

const notifications = useNotifications()
const revisions = ref<any[]>([])
const loading = ref(false)

const namespace = computed(() => itemNs(props.resource))
const name = computed(() => itemName(props.resource))

const sorted = computed(() =>
  [...revisions.value].sort((a, b) => parseInt(String(b.revision), 10) - parseInt(String(a.revision), 10)),
)

async function load() {
  loading.value = true
  try {
    revisions.value = (await callApp('GetDeploymentRevisions', namespace.value, name.value)) ?? []
  } catch (e: any) {
    revisions.value = []
    notifications.notifyError(`Ревизии: ${e?.message || e}`)
  } finally {
    loading.value = false
  }
}

async function rollback(revision: string | number) {
  if (!(await askConfirm(`Rollback ${name.value} to revision ${revision}?`))) return
  try {
    await callApp('RollbackDeployment', namespace.value, name.value, parseInt(String(revision), 10))
    notifications.notifySuccess(`${name.value}: rollback to revision ${revision} started`)
    await load()
    await reloadActive()
  } catch (e: any) {
    notifications.notifyError(`Rollback: ${e?.message || e}`)
  }
}

watch(() => `${namespace.value}/${name.value}`, load, { immediate: true })
</script>
