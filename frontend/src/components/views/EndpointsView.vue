<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-gray-700">
      <h2 class="text-xl font-semibold">Endpoints</h2>
    </div>
    <div class="flex-1 overflow-auto p-4">
      <div v-if="loading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
      <div v-else-if="error" class="text-red-500 p-4">
        {{ error }}
      </div>
      <div v-else-if="endpoints.length === 0" class="text-gray-400 p-4">
        No Endpoints found
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="ep in endpoints"
          :key="`${ep.namespace}/${ep.name}`"
          class="bg-gray-800 rounded p-3 cursor-pointer hover:bg-gray-700 transition"
        >
          <div class="flex justify-between items-start">
            <div>
              <div class="font-semibold text-white">{{ ep.name }}</div>
              <div class="text-sm text-gray-400">{{ ep.namespace }}</div>
            </div>
            <div class="text-right text-sm">
              <div class="text-blue-400">{{ ep.addresses }} addresses</div>
              <div class="text-gray-500">{{ ep.age }}</div>
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface Endpoint {
  name: string
  namespace: string
  addresses: number
  ports: number
  age: string
}

const endpoints = ref<Endpoint[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const loadEndpoints = async () => {
  loading.value = true
  error.value = null
  try {
    endpoints.value = []
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load Endpoints'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadEndpoints()
})
</script>
