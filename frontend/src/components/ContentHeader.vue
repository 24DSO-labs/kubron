<template>
  <div class="content-header">
    <h2>{{ icon }} {{ title }}</h2>
    <span class="count" v-if="count !== undefined && count >= 0">{{ count }} items</span>
    <input v-if="showSearch" :value="search" @input="$emit('search', ($event.target as HTMLInputElement).value)" placeholder="Поиск..." class="search-input"/>
    <button @click="$emit('refresh')" class="btn btn-sm">🔄</button>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ icon: string; title: string; count?: number; search?: string; showSearch?: boolean }>()
defineEmits<{ (e: 'refresh'): void; (e: 'search', val: string): void }>()
</script>