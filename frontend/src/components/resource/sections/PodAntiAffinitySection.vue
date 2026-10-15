<template>
  <CodeBlockSection title="Pod Anti Affinities" :lines="lines" />
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import CodeBlockSection from './CodeBlockSection.vue'
import { buildPodAntiAffinityYaml } from '../../../resources/yaml'

const props = defineProps<{ resource: any }>()
const lines = computed(() => buildPodAntiAffinityYaml(props.resource?.podAntiAffinity))
</script>
