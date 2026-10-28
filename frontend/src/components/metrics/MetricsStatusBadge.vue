<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <span :class="['status-badge', badgeClasses]">
    <span :class="['status-badge-dot', dotClass]"></span>
    {{ label }}
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'

const props = defineProps<{
  status: string
}>()
const { t } = useI18n()

const label = computed(() => {
  switch (props.status) {
    case 'installed': return t('metrics.status.active')
    case 'not_installed': return t('metrics.status.notInstalled')
    case 'degraded': return t('metrics.status.degraded')
    case 'installing': return t('metrics.status.installing')
    default: return t('metrics.status.unknown')
  }
})

const badgeClasses = computed(() => {
  switch (props.status) {
    case 'installed': return 'bg-green-900/50 text-green-300 border border-green-700/50'
    case 'not_installed': return 'bg-gray-700/50 text-gray-400 border border-gray-600/50'
    case 'degraded': return 'bg-yellow-900/50 text-yellow-300 border border-yellow-700/50'
    case 'installing': return 'bg-blue-900/50 text-blue-300 border border-blue-700/50'
    default: return 'bg-gray-700/50 text-gray-400 border border-gray-600/50'
  }
})

const dotClass = computed(() => {
  switch (props.status) {
    case 'installed': return 'bg-green-400 animate-pulse'
    case 'not_installed': return 'bg-gray-500'
    case 'degraded': return 'bg-yellow-400 animate-pulse'
    case 'installing': return 'bg-blue-400 animate-spin'
    default: return 'bg-gray-500'
  }
})
</script>