<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <CodeBlockSection :title="$t('sections.podAntiAffinity.title')" :lines="lines" />
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import CodeBlockSection from './CodeBlockSection.vue'
import { buildPodAntiAffinityYaml } from '../../../resources/yaml'

const props = defineProps<{ resource: any }>()
const lines = computed(() => buildPodAntiAffinityYaml(props.resource?.podAntiAffinity))
</script>
