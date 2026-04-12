<template>
  <div>
    <ResourceListBase
      :items="itemsWithMetrics"
      :search="search"
      resource-title="Pods"
      empty-message="Нет подов"
      resource-type="pod"
      :custom-columns="columns"
      @refresh="refreshData"
      @search="(v) => search = v"
      @select="onSelect"
      @contextAction="onContextAction"
    />
    
    <!-- Dialog подтверждения Evict -->
    <ConfirmationDialog
      v-model="showEvictDialog"
      resource-type="pod"
      :resource-name="podToEvict?.name || ''"
      :namespace="podToEvict?.namespace"
      :title="`Evict pod ${podToEvict?.name}?`"
      @confirm="confirmEvict"
    />
  </div>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import { useK8s } from '../../../composables/useK8s'
import useResourceView from '../../../composables/useResourceView'
import ResourceListBase from '../ResourceListBase.vue'
import ConfirmationDialog from '../../utils/ConfirmationDialog.vue'
import type { PodInfo, Column } from '../../../types'
import { computed } from 'vue'
import { EvictPod } from '../../../../wailsjs/go/main/App'


const { filteredPods, refreshData, search, mergePodMetrics, podMetrics } = useK8s()

const { onSelect, onContextAction: baseOnContextAction } = useResourceView('pod')

// Состояние для диалога Evict
const showEvictDialog = ref(false)
const podToEvict = ref<any>(null)

// Кастомный обработчик контекстного меню для подов с поддержкой evict
const onContextAction = (action: string, item: any) => {
  if (action === 'evict') {
    podToEvict.value = item
    showEvictDialog.value = true
    return
  }
  // Остальные действия передаём в базовый обработчик
  baseOnContextAction(action, item)
}

// Подтверждение Evict
const confirmEvict = () => {
  if (!podToEvict.value) return
  
  EvictPod(podToEvict.value.namespace || '', podToEvict.value.name)
    .then(() => {
      console.log(`Pod ${podToEvict.value.name} evicted`)
      refreshData()
    })
    .catch((err: any) => {
      console.error('Error evicting pod:', err)
    })
    .finally(() => {
      podToEvict.value = null
    })
}

const itemsWithMetrics = computed(() => {
  return mergePodMetrics(filteredPods.value, podMetrics.value)
})

const columns: Column[] = [
  { key: 'name', label: 'NAME', className: 'resource-name' },
  { key: 'namespace', label: 'NAMESPACE', className: 'ns-label' },
  { key: 'cpu', label: 'CPU', className: 'mono metric-cell' },
  { key: 'memory', label: 'MEMORY', className: 'mono metric-cell' },
  { 
    key: 'containers', 
    label: 'CONTAINERS',
    formatter: (item) => {
      if (!item?.containers) return ''
      return item.containers.map((c: any) => {
        let color = 'red'
        if (c.isInit) color = 'gray'
        else if (c.ready) color = 'green'
        else if (c.state?.includes('wait') || c.state?.includes('Creat') || c.state?.includes('Pend')) color = 'yellow'
        else if (c.state === 'terminated' || c.state?.includes('Complet')) color = 'gray'
        return `<span class="container-square square-${color}" title="${c.name}"></span>`
      }).join('')
    }
  },
  { key: 'status', label: 'STATUS', className: 'status-label' },
  { key: 'restarts', label: 'RESTARTS' },
  { key: 'age', label: 'AGE' },
  { key: 'node', label: 'NODE' },
  { key: 'qos', label: 'QOS' }
]

</script>

<style scoped>
.container-square {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 2px;
  cursor: help;
}

.square-green {
  background-color: #4ade80;
}

.square-red {
  background-color: #ef4444;
}

.square-yellow {
  background-color: #facc15;
}

.square-gray {
  background-color: #868d8a;
}
</style>
