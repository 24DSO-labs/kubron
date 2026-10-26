<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <CodeBlockSection title="Node Affinities" :lines="lines" />
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import CodeBlockSection from './CodeBlockSection.vue'
import { buildNodeAffinityYaml } from '../../../resources/yaml'

const props = defineProps<{ resource: any }>()
const lines = computed(() => buildNodeAffinityYaml(props.resource?.affinity))
</script>
