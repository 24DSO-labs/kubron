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

      <template v-for="entry in navEntries" :key="entry.type === 'group' ? entry.id : entry.view">
        <div
          v-if="entry.type === 'item'"
          :class="['nav-item', { active: currentView === entry.view }]"
          @click="emit('navigate', entry.view)"
        >
          <span class="nav-icon"></span> {{ entry.label }}
        </div>

        <div v-else class="nav-group">
          <div class="nav-group-header" @click="toggle(entry.id)">
            <span class="nav-arrow" :class="{ open: expanded[entry.id] }">›</span>
            <span class="nav-group-icon"></span>
            <span class="nav-group-title">{{ entry.title }}</span>
          </div>
          <div v-show="expanded[entry.id]" class="nav-group-items">
            <div
              v-for="item in entry.items"
              :key="item.view"
              :class="['nav-item', { active: currentView === item.view }]"
              @click="emit('navigate', item.view)"
            >
              <span class="nav-icon"></span> {{ item.label }}
            </div>
          </div>
        </div>
      </template>

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
import { reactive } from 'vue'
import type { NamespaceInfo } from '../types'
import type { NavGroupId } from '../resources/types'
import { NAV_GROUPS, buildNav } from '../resources/navigation'
import { can } from '../composables/useAuth'
import { Perm } from '../types/auth'

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

const expanded = reactive(
  Object.fromEntries(Object.entries(NAV_GROUPS).map(([id, g]) => [id, g.expanded])) as Record<NavGroupId, boolean>,
)

function toggle(group: NavGroupId) {
  expanded[group] = !expanded[group]
}
</script>
