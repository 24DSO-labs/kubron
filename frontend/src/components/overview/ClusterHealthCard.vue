<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card health">
    <h2 class="ov-card-title">{{ $t('overview.health.title') }}</h2>

    <div class="health-body">
      <div class="health-ring" :class="`tone-${tone}`" role="img" :aria-label="$t('overview.health.score', { score: health.score })">
        <svg viewBox="0 0 100 100" aria-hidden="true">
          <circle class="health-ring-track" cx="50" cy="50" :r="RADIUS" />
          <circle
            class="health-ring-value"
            cx="50" cy="50" :r="RADIUS"
            :stroke-dasharray="CIRCUMFERENCE"
            :stroke-dashoffset="offset"
            transform="rotate(-90 50 50)"
          />
        </svg>
        <div class="health-ring-label">
          <span class="health-ring-score">{{ health.score }}</span>
          <span class="health-ring-status">{{ statusText }}</span>
        </div>
      </div>

      <ul class="components">
        <li v-for="c in health.components" :key="c.name">
          <span class="c-name">{{ c.name }}</span>
          <span class="c-status" :class="`s-${c.status}`">{{ componentStatus(c.status) }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import type { ClusterHealth, HealthStatus } from '../../types/overview'

const props = defineProps<{ health: ClusterHealth }>()

const RADIUS = 42
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

const { t, te } = useI18n()

const componentStatus = (s: HealthStatus) => (te(`overview.health.component.${s}`) ? t(`overview.health.component.${s}`) : s)
// Статус кластера — значение API (Healthy | Degraded | Critical); неизвестное показываем как есть
const statusText = computed(() => {
  const s = props.health.status
  return te(`overview.health.status.${s}`) ? t(`overview.health.status.${s}`) : s
})

const offset = computed(() => {
  const score = Math.min(Math.max(props.health.score, 0), 100)
  return CIRCUMFERENCE * (1 - score / 100)
})

const tone = computed(() => {
  if (props.health.score >= 90) return 'ok'
  if (props.health.score >= 70) return 'warn'
  return 'danger'
})
</script>

<style scoped>
.health-body {
  display: flex;
  align-items: center;
  flex-wrap: wrap; /* в узкой карточке список компонентов уходит под кольцо */
  gap: 16px 24px;
  flex: 1;
  min-height: 0;
}

.health-ring {
  position: relative;
  width: 132px;
  height: 132px;
  flex-shrink: 0;
}
.health-ring svg { width: 100%; height: 100%; display: block; }
.health-ring-track { fill: none; stroke: var(--ov-border); stroke-width: 8; }
.health-ring-value {
  fill: none;
  stroke: var(--ov-ok);
  stroke-width: 8;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.6s ease, stroke 0.3s;
}
.tone-warn .health-ring-value { stroke: var(--ov-warn); }
.tone-danger .health-ring-value { stroke: var(--ov-danger); }

.health-ring-label {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.health-ring-score { font-size: 34px; font-weight: 600; line-height: 1; font-variant-numeric: tabular-nums; }
.health-ring-status { margin-top: 4px; font-size: 12px; color: var(--ov-muted); }

.components { list-style: none; flex: 1 1 150px; min-width: 0; display: flex; flex-direction: column; gap: 9px; }
.components li { display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 13px; }
.c-name { color: var(--ov-text); }
.c-status { font-size: 12px; font-weight: 500; }
.s-healthy { color: var(--ov-ok); }
.s-unhealthy { color: var(--ov-danger); }
.s-unknown { color: var(--ov-muted); }
</style>
