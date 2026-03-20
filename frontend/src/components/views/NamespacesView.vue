<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-gray-700">
      <h2 class="text-xl font-semibold">Namespaces</h2>
    </div>
    <div class="flex-1 overflow-auto p-4">
      <div v-if="loading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
      <div v-else-if="error" class="text-red-500 p-4">
        {{ error }}
      </div>
      <div v-else-if="displayNamespaces.length === 0" class="text-gray-400 p-4">
        No Namespaces found
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="ns in displayNamespaces"
          :key="ns.name"
          class="bg-gray-800 rounded p-3 cursor-pointer hover:bg-gray-700 transition"
        >
          <div class="flex justify-between items-start">
            <div>
              <div class="font-semibold text-white">{{ ns.name }}</div>
              <div class="text-sm text-gray-400">{{ ns.status }}</div>
            </div>
            <div class="text-right text-sm text-gray-500">
              {{ ns.createdAt }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, computed } from 'vue'
import { useK8s } from '../../composables/useK8s'

const { namespaces: allNamespaces } = useK8s()

const loading = ref(false)
const error = ref<string | null>(null)

const displayNamespaces = computed(() => {
  return (allNamespaces.value || []).map(ns => ({
    name: ns.name,
    status: 'Active',
    createdAt: new Date().toLocaleDateString()
  }))
})
</script>
