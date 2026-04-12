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
  </div>
</template>

<script lang="ts" setup>
import { useK8s } from '../../../composables/useK8s'
import useResourceView from '../../../composables/useResourceView'
import ResourceListBase from '../ResourceListBase.vue'
import type { PodInfo, Column } from '../../../types'
import { computed } from 'vue'


const { filteredPods, refreshData, search, mergePodMetrics, podMetrics } = useK8s()

const { onSelect, onContextAction } = useResourceView('pod')

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
