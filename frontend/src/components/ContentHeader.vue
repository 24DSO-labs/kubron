<template>
  <div class="content-header">
    <h2>
      <K8sIcon v-if="icon" :name="icon" size="lg" class="header-icon" />
      {{ title }}
    </h2>
    <span class="count" v-if="count !== undefined && count >= 0">{{ count }} items</span>
    <input v-if="showSearch" :value="search" @input="$emit('search', ($event.target as HTMLInputElement).value)" placeholder="Поиск..." class="search-input"/>
    <button @click="$emit('refresh')" class="btn btn-sm" title="Refresh">
      <K8sIcon name="refresh" size="sm" />
    </button>
  </div>
</template>

<script lang="ts" setup>
import K8sIcon from './icons/K8sIcon.vue'

defineProps<{ icon?: string; title: string; count?: number; search?: string; showSearch?: boolean }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>

<style scoped>
.header-icon {
  margin-right: 8px;
  vertical-align: middle;
}
</style>