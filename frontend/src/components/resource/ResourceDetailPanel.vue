<template>
  <K8sResourceDetailBase
    :resource="base"
    :resource-icon="def.icon"
    :resource-type="def.kind"
    :custom-tabs="tabList"
    :show-edit="def.editable !== false"
    :show-delete="!!def.api.remove"
    @close="$emit('close')"
    @edit="resourceActions.edit(def.id, ns, name)"
    @delete="resourceActions.requestDelete(def.id, ns, name)"
    @namespaceClick="onNamespaceClick"
  >
    <!-- Edit / Delete / Close рисует базовый компонент; здесь только специфичные действия -->
    <template #actions>
      <template v-for="a in visibleActions" :key="a.id">
        <!-- Действия с dropdown по контейнерам (attach/shell/logs) -->
        <div v-if="a.perContainer && containers.length > 1" class="rd-action-group" data-rd-dropdown>
          <button
            class="rd-action"
            :class="{ danger: a.danger }"
            :title="a.label"
            @click="toggleDropdown(a.id)"
          >
            <K8sIcon v-if="a.icon" :name="a.icon" :alt="a.label" size="lg" :danger="a.danger" />
            <template v-else>{{ a.label }}</template>
          </button>
          <div v-if="openDropdown === a.id" class="rd-action-dropdown">
            <div
              v-for="c in containers"
              :key="c.name"
              class="rd-dropdown-item"
              @click="runContainerAction(a, c.name)"
            >
              {{ c.name }}
            </div>
          </div>
        </div>
        <!-- Обычное действие (1 контейнер или не контейнерное) -->
        <button
          v-else
          class="rd-action"
          :class="{ danger: a.danger }"
          :title="a.label"
          @click="a.run(resource, resourceActions)"
        >
          <K8sIcon v-if="a.icon" :name="a.icon" :alt="a.label" size="lg" :danger="a.danger" />
          <template v-else>{{ a.label }}</template>
        </button>
      </template>
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
          <span v-else-if="p.link" class="rd-value">
            <a
              v-if="p.link(resource)"
              class="rd-link"
              @click="resourceActions.open(p.link(resource)!.id, p.link(resource)!.ns, p.link(resource)!.name)"
            >
              {{ show(p.value(resource)) }}
            </a>
            <template v-else>{{ show(p.value(resource)) }}</template>
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
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import K8sResourceDetailBase from '../panels/K8sResourceDetailBase.vue'
import K8sIcon from '../icons/K8sIcon.vue'
import KeyValueSection from './sections/KeyValueSection.vue'
import type { ActionDef, ResourceDefinition } from '../../resources/types'
import { resourceActions } from '../../resources/actions'
import { containersOf, itemName, itemNs } from '../../resources/utils'

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

// ─── Per-container dropdown (ActionDef.perContainer) ───
const containers = computed(() => containersOf(props.resource))
const openDropdown = ref<string | null>(null)

function toggleDropdown(id: string) {
  openDropdown.value = openDropdown.value === id ? null : id
}

function runContainerAction(a: ActionDef, container: string) {
  openDropdown.value = null
  return a.run(props.resource, resourceActions, container)
}

// Закрывать список по клику вне него
function closeDropdown(e: MouseEvent) {
  if (!(e.target as Element | null)?.closest?.('[data-rd-dropdown]')) openDropdown.value = null
}
onMounted(() => document.addEventListener('click', closeDropdown))
onBeforeUnmount(() => document.removeEventListener('click', closeDropdown))
</script>
