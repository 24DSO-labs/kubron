<template>
  <div class="containers-display" v-html="getContainersHtml"></div>
</template>

<script lang="ts" setup>
import type { PodInfo } from '../../types'
import { computed } from 'vue'

interface Props {
  item: PodInfo
  col: any
}

const props = defineProps<Props>()

const getContainersHtml = computed(() => {
  const pod = props.item
  if (!pod?.containers) return ''
  
  let html = ''
  for (const c of pod.containers) {
    let color = 'red'
    if (c.isInit) {
      color = 'gray'
    } else if (c.ready) {
      color = 'green'
    } else if (c.state.includes('wait') || c.state.includes('Creat') || c.state.includes('Pend') || c.state === 'waiting') {
      color = 'yellow'
    } else if (c.state === 'terminated' || c.state.includes('Complet')) {
      color = 'gray'
    }
    html += `<span class="container-square square-${color}" title="${c.name}"></span>`
  }
  return html
})
</script>

<style scoped>
.containers-display {
  display: flex;
  gap: 8px;
  align-items: center;
  flex-wrap: wrap;
}

.container-group {
  display: flex;
  gap: 4px;
  align-items: center;
}

.container-square {
  display: inline-block;
  width: 12px;
  height: 12px;
  border-radius: 2px;
  cursor: help;
}

.init-container {
  opacity: 0.6;
  border: 1px solid currentColor;
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
