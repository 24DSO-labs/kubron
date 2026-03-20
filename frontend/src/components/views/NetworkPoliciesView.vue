<template>
  <div class="flex flex-col h-full">
    <div class="px-4 py-3 border-b border-gray-700">
      <h2 class="text-xl font-semibold">Network Policies</h2>
    </div>
    <div class="flex-1 overflow-auto p-4">
      <div v-if="loading" class="flex items-center justify-center h-full">
        <div class="animate-spin rounded-full h-12 w-12 border-b-2 border-blue-500"></div>
      </div>
      <div v-else-if="error" class="text-red-500 p-4">
        {{ error }}
      </div>
      <div v-else-if="networkPolicies.length === 0" class="text-gray-400 p-4">
        No Network Policies found
      </div>
      <div v-else class="space-y-2">
        <div
          v-for="np in networkPolicies"
          :key="`${np.namespace}/${np.name}`"
          class="bg-gray-800 rounded p-3 cursor-pointer hover:bg-gray-700 transition"
        >
          <div class="flex justify-between items-start">
            <div>
              <div class="font-semibold text-white">{{ np.name }}</div>
              <div class="text-sm text-gray-400">{{ np.namespace }}</div>
            </div>
            <div class="text-right text-sm text-gray-500">
              {{ np.age }}
            </div>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'

interface NetworkPolicy {
  name: string
  namespace: string
  podSelector: string
  age: string
}

const networkPolicies = ref<NetworkPolicy[]>([])
const loading = ref(false)
const error = ref<string | null>(null)

const loadNetworkPolicies = async () => {
  loading.value = true
  error.value = null
  try {
    networkPolicies.value = []
  } catch (err) {
    error.value = err instanceof Error ? err.message : 'Failed to load Network Policies'
  } finally {
    loading.value = false
  }
}

onMounted(() => {
  loadNetworkPolicies()
})
</script>
