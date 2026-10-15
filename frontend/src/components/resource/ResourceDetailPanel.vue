<template>
  <K8sResourceDetailBase
    :resource="base"
    :resource-icon="def.icon"
    :custom-tabs="tabList"
    @close="$emit('close')"
    @edit="resourceActions.edit(def.id, ns, name)"
    @delete="resourceActions.requestDelete(def.id, ns, name)"
    @namespaceClick="onNamespaceClick"
  >
    <!-- Edit / Delete / Close рисует базовый компонент; здесь только специфичные действия -->
    <template #actions>
      <button
        v-for="a in visibleActions"
        :key="a.id"
        class="rd-action"
        :class="{ danger: a.danger }"
        :title="a.label"
        @click="a.run(resource, resourceActions)"
      >
        <img v-if="a.icon" :src="a.icon" :alt="a.label" />
        <template v-else>{{ a.label }}</template>
      </button>
    </template>

    <template #properties>
      <div class="rd-props">
        <component :is="s" v-for="(s, i) in def.headSections" :key="'h' + i" :resource="resource" @open="onOpen" />

        <KeyValueSection title="Labels" :data="labels" />
        <KeyValueSection title="Annotations" :data="annotations" />

        <div v-for="p in def.properties" :key="p.label" class="rd-row">
          <span class="rd-label">{{ p.label }}</span>
          <span v-if="p.badge" class="rd-value">
            <span class="rd-badge" :class="p.badge(resource)">{{ show(p.value(resource)) }}</span>
          </span>
          <span v-else class="rd-value">{{ show(p.value(resource)) }}</span>
        </div>

        <component :is="s" v-for="(s, i) in def.sections" :key="'s' + i" :resource="resource" @open="onOpen" />
      </div>
    </template>

    <!-- Дополнительные вкладки из определения ресурса -->
    <template v-for="t in tabList" #[t.slot]>
      <component :is="tabComponents[t.id]" :resource="resource" @open="onOpen" />
    </template>

    <template #events>
      <div class="rd-empty">No events available</div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import './resource.css'
import { computed } from 'vue'
import K8sResourceDetailBase from '../panels/K8sResourceDetailBase.vue'
import KeyValueSection from './sections/KeyValueSection.vue'
import type { ResourceDefinition } from '../../resources/types'
import { resourceActions } from '../../resources/actions'
import { itemName, itemNs } from '../../resources/utils'

const props = defineProps<{
  def: ResourceDefinition
  resource: any
  ns: string
  name: string
}>()
defineEmits<{ (e: 'close'): void }>()

/** Базовый компонент читает resource.name / namespace / age / created: приводим к плоскому виду */
const base = computed(() => {
  const r = props.resource ?? {}
  return {
    ...r,
    name: props.name || itemName(r),
    namespace: props.def.namespaced ? props.ns || itemNs(r) : '',
    age: r.age ?? '',
    created: r.created ?? r.createdTime ?? r.metadata?.creationTimestamp ?? '',
  }
})

const labels = computed(() => props.resource?.labels ?? props.resource?.metadata?.labels)
const annotations = computed(() => props.resource?.annotations ?? props.resource?.metadata?.annotations)

const visibleActions = computed(() =>
  (props.def.actions ?? []).filter(a => !a.when || a.when(props.resource)),
)

const tabList = computed(() =>
  (props.def.tabs ?? []).map(t => ({ id: t.id, label: t.label, slot: `tab-${t.id}` })),
)
const tabComponents = computed(() =>
  Object.fromEntries((props.def.tabs ?? []).map(t => [t.id, t.component])),
)

const show = (v: unknown) => (v === undefined || v === null || v === '' ? '—' : v)

/** Секции и вкладки открывают другие ресурсы через событие open */
function onOpen(p: { id: string; ns: string; name: string; item?: any }) {
  resourceActions.open(p.id, p.ns, p.name, p.item)
}

function onNamespaceClick(namespace: string) {
  resourceActions.open('namespace', '', namespace)
}
</script>
