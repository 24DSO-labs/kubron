<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card problems">
    <header class="head">
      <h2 class="ov-card-title">{{ $t('overview.problems.title') }}</h2>
      <span v-if="total" class="count">{{ total }}</span>
    </header>

    <ul v-if="pods.length" class="list">
      <li v-for="p in pods" :key="`${p.namespace}/${p.name}`">
        <button type="button" class="row" @click="emit('open', p)">
          <span class="name" :title="p.name">{{ p.name }}</span>
          <span class="tag" :class="tone(p.reason)">{{ p.reason }}</span>
          <span class="sub">
            {{ p.namespace }}<template v-if="p.node"> · {{ p.node }}</template>
            <template v-if="p.restarts"> · {{ $t('overview.problems.restarts', p.restarts) }}</template>
            · {{ timeAgo(p.since, now) }}
          </span>
        </button>
      </li>
    </ul>
    <p v-else class="none">{{ $t('overview.problems.empty') }}</p>

    <button v-if="total > pods.length" type="button" class="more" @click="emit('viewAll')">
      {{ $t('overview.problems.viewAll', { total }) }}
    </button>
  </section>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { timeAgo } from './format'
import type { ProblemPod } from '../../types/overview'

defineProps<{ pods: ProblemPod[]; total: number }>()
const emit = defineEmits<{ (e: 'open', p: ProblemPod): void; (e: 'viewAll'): void }>()

const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | null = null
onMounted(() => { tick = setInterval(() => (now.value = Date.now()), 30_000) })
onUnmounted(() => { if (tick) clearInterval(tick) })

const SEVERE = ['CrashLoopBackOff', 'RunContainerError', 'Failed', 'Evicted', 'OOMKilled']
const tone = (reason: string) => (SEVERE.includes(reason) ? 't-err' : 't-warn')
</script>

<style scoped>
.head { display: flex; align-items: center; gap: 10px; margin-bottom: 12px; }
.head .ov-card-title { margin: 0; }
.count { font-size: 12px; color: var(--ov-warn); }
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 2px; max-height: 330px; overflow: auto; }
.row {
  display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'name tag' 'sub sub';
  gap: 2px 8px; align-items: center; width: 100%; padding: 7px 8px; background: transparent;
  border: 0; border-radius: 8px; font: inherit; text-align: left; color: var(--ov-text); cursor: pointer;
}
.row:hover, .row:focus-visible { background: var(--ov-card-hover); outline: none; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.tag { grid-area: tag; padding: 1px 8px; border-radius: 999px; font-size: 11px; font-weight: 600; }
.t-err { color: var(--ov-danger); background: rgba(248, 113, 113, 0.12); }
.t-warn { color: var(--ov-warn); background: rgba(251, 191, 36, 0.12); }
.none { margin: 0; font-size: 13px; color: var(--ov-ok); }
.more { align-self: flex-start; margin-top: 10px; padding: 0; background: none; border: 0; font: inherit; font-size: 12px; color: var(--ov-accent); cursor: pointer; }
.more:hover { text-decoration: underline; }
</style>
