<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-gray-700 flex justify-between items-center">
      <div>
        <h2 class="text-xl font-semibold">⭐ Cluster Roles</h2>
        <p class="text-sm text-gray-400">{{ items.length }} items</p>
      </div>
      <input
        :value="search"
        @input="onSearch(($event.target as HTMLInputElement).value)"
        type="text"
        placeholder="Поиск..."
        class="px-3 py-1 bg-gray-700 border border-gray-600 rounded text-sm"
      />
    </div>
    <div class="flex-1 overflow-auto p-4">
      <div v-if="items.length === 0" class="text-gray-400 p-4">
        Нет Cluster Roles
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="cr in items"
          :key="cr.metadata?.name"
          class="bg-gray-800 rounded p-3 cursor-pointer hover:bg-gray-700 transition"
        >
          <div class="flex justify-between items-start">
            <div class="flex-1">
              <div class="font-semibold text-white">{{ cr.metadata?.name }}</div>
              <div class="text-xs text-gray-500 mt-1">Rules: {{ cr.rules?.length || 0 }}</div>
            </div>
            <div class="text-right text-sm text-gray-500">
              {{ getAge(cr.metadata?.creationTimestamp) }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { ClusterRole } from '../../types'

defineProps<{
  items: ClusterRole[]
  search: string
}>()

const emit = defineEmits<{
  (e: 'refresh'): void
  (e: 'search', val: string): void
}>()

function onSearch(val: string) { emit('search', val) }

function getAge(creationTimestamp: string | undefined): string {
  if (!creationTimestamp) return '-'
  const date = new Date(creationTimestamp)
  const now = new Date()
  const diff = now.getTime() - date.getTime()
  const days = Math.floor(diff / (1000 * 60 * 60 * 24))
  const hours = Math.floor((diff % (1000 * 60 * 60 * 24)) / (1000 * 60 * 60))
  if (days > 0) return `${days}d`
  return `${hours}h`
}
</script>
