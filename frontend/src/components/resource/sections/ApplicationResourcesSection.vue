<template>
  <div class="rd-section">
    <div class="rd-section-title">Workload Resources</div>
    <ResourcesTable :rows="workloads" kinds-clickable @open="emit('open', $event)" />
  </div>
  <div class="rd-section">
    <div class="rd-section-title">Other Resources</div>
    <ResourcesTable :rows="others" @open="emit('open', $event)" />
  </div>
</template>

<script lang="ts" setup>
import { computed, defineComponent, h, type PropType } from 'vue'

/** Workload-ы и прочие ресурсы приложения (Name / Kind / Component, как в Lens) */
const props = defineProps<{ resource: any }>()
const emit = defineEmits<{
  (e: 'open', p: { id: string; ns: string; name: string; item?: any }): void
}>()

interface Row { kind: string; name: string; component?: string }

const workloads = computed<Row[]>(() => props.resource?.workloads ?? [])
const others = computed<Row[]>(() => props.resource?.resources ?? [])

/** Kind -> id ресурса в приложении; для остальных строка не кликабельна */
const KIND_IDS: Record<string, string> = {
  Deployment: 'deployment', StatefulSet: 'statefulset', DaemonSet: 'daemonset',
  Service: 'service', Ingress: 'ingress', ConfigMap: 'configmap', ServiceAccount: 'serviceaccount',
  PersistentVolumeClaim: 'persistentvolumeclaim', Role: 'role', RoleBinding: 'rolebinding',
}

const ResourcesTable = defineComponent({
  props: {
    rows: { type: Array as PropType<Row[]>, required: true },
    kindsClickable: Boolean,
  },
  emits: ['open'],
  setup(p, { emit: e }) {
    return () => {
      if (!p.rows.length) return h('div', { class: 'rd-empty' }, 'No resources available')
      const ns = props.resource?.namespace ?? ''
      return h('div', { class: 'rd-table-wrap' }, h('table', { class: 'rd-table' }, [
        h('thead', h('tr', ['Name', 'Kind', 'Component'].map(c => h('th', c)))),
        h('tbody', p.rows.map(r => {
          const id = KIND_IDS[r.kind]
          return h('tr', {
            key: `${r.kind}/${r.name}`,
            class: id ? 'rd-click' : '',
            onClick: id ? () => e('open', { id, ns, name: r.name }) : undefined,
          }, [
            h('td', { class: id ? 'rd-link' : '' }, r.name),
            h('td', r.kind),
            h('td', { class: 'rd-muted' }, r.component || '—'),
          ])
        })),
      ]))
    }
  },
})
</script>
