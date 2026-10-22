<template>
  <aside class="sidebar">
    <!-- Cluster name -->
    <div class="sidebar-cluster-name" v-if="clusterName">
      <span class="cluster-name-text">{{ clusterName }}</span>
    </div>

    <!-- Navigation Tree: пункты строятся из resources/navigation.ts -->
    <nav class="nav-tree">
      <div :class="['nav-item', { active: currentView === 'overview' }]" @click="emit('navigate', 'overview')">
        <span class="nav-icon"></span> Overview
      </div>

      <template v-for="entry in visibleEntries" :key="entry.type === 'group' ? entry.id : entry.view">
        <div
          v-if="entry.type === 'item'"
          :class="['nav-item', { active: currentView === entry.view }]"
          @click="emit('navigate', entry.view)"
        >
          <span class="nav-icon"></span> {{ entry.label }}
        </div>

        <NavGroupNode
          v-else
          :group="entry"
          :expanded="expanded"
          :current-view="currentView"
          :available="available"
          @toggle="toggle"
          @navigate="emit('navigate', $event)"
        />
      </template>
 <CustomResourcesNav :current-view="currentView" @navigate="emit('navigate', $event)" />

      <!-- Settings -->
      <div class="nav-separator"></div>
      <div :class="['nav-item', { active: currentView === 'settings' }]" @click="emit('navigate', 'settings')">
        <span class="nav-icon">⚙️</span> Settings
      </div>
      <div v-if="can(Perm.UsersManage)" :class="['nav-item', { active: currentView === 'users' }]"
        @click="emit('navigate', 'users')">
        <span class="nav-icon">👥</span> Users
      </div>
      <div v-if="can(Perm.AuditRead)" :class="['nav-item', { active: currentView === 'audit' }]"
        @click="emit('navigate', 'audit')">
        <span class="nav-icon">📜</span> Audit
      </div>
    </nav>

    <!-- Namespace selector -->
    <div class="ns-selector">
      <label class="ns-label-text">Namespace</label>
      <select
        :key="selectedNamespace"
        :value="selectedNamespace"
        @change="emit('nsChange', ($event.target as HTMLSelectElement).value)"
        class="ns-select"
      >
        <option value="">All namespaces</option>
        <option v-for="ns in namespaces" :key="ns.name" :value="ns.name">{{ ns.name }}</option>
      </select>
    </div>
  </aside>
</template>

<script lang="ts" setup>
import { computed, reactive } from 'vue'
import type { NamespaceInfo } from '../types'
import type { NavGroupId } from '../resources/types'
import { NAV_GROUPS, buildNav } from '../resources/navigation'
import NavGroupNode from './nav/NavGroupNode.vue'
import { useArgo } from '../composables/useArgo'
import { can } from '../composables/useAuth'
import { Perm } from '../types/auth'
import CustomResourcesNav from './nav/CustomResourcesNav.vue'

defineProps<{
  currentView: string
  selectedNamespace: string
  namespaces: NamespaceInfo[]
  clusterName?: string
}>()

const emit = defineEmits<{
  (e: 'navigate', view: string): void
  (e: 'nsChange', ns: string): void
}>()

const navEntries = buildNav()

// Условия `requires` у групп меню: сейчас только наличие Argo CD в кластере
const { argoInstalled } = useArgo()
const available = computed(() => new Set<string>(argoInstalled.value ? ['argocd'] : []))
const visibleEntries = computed(() =>
  navEntries.filter(e => e.type === 'item' || !e.requires || available.value.has(e.requires)),
)

const expanded = reactive(
  Object.fromEntries(Object.entries(NAV_GROUPS).map(([id, g]) => [id, g.expanded])) as Record<NavGroupId, boolean>,
)

function toggle(group: NavGroupId) {
  expanded[group] = !expanded[group]
}
</script>
