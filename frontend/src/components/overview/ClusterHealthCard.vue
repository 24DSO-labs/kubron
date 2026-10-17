<template>
  <section class="ov-card health">
    <h2 class="ov-card-title">Cluster health</h2>

    <div class="health-body">
      <div class="ring" :class="`tone-${tone}`" role="img" :aria-label="`Health score ${health.score} of 100`">
        <svg viewBox="0 0 100 100" aria-hidden="true">
          <circle class="ring-track" cx="50" cy="50" :r="RADIUS" />
          <circle
            class="ring-value"
            cx="50" cy="50" :r="RADIUS"
            :stroke-dasharray="CIRCUMFERENCE"
            :stroke-dashoffset="offset"
            transform="rotate(-90 50 50)"
          />
        </svg>
        <div class="ring-label">
          <span class="ring-score">{{ health.score }}</span>
          <span class="ring-status">{{ health.status }}</span>
        </div>
      </div>

      <ul class="components">
        <li v-for="c in health.components" :key="c.name">
          <span class="c-name">{{ c.name }}</span>
          <span class="c-status" :class="`s-${c.status}`">{{ STATUS_LABEL[c.status] }}</span>
        </li>
      </ul>
    </div>
  </section>
</template>

<script lang="ts" setup>
import { computed } from 'vue'
import type { ClusterHealth, HealthStatus } from '../../types/overview'

const props = defineProps<{ health: ClusterHealth }>()

const RADIUS = 42
const CIRCUMFERENCE = 2 * Math.PI * RADIUS

const STATUS_LABEL: Record<HealthStatus, string> = {
  healthy: 'Healthy',
  unhealthy: 'Unhealthy',
  unknown: 'Unknown',
}

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

.ring { position: relative; width: 132px; height: 132px; flex-shrink: 0; }
.ring svg { width: 100%; height: 100%; display: block; }
.ring-track { fill: none; stroke: var(--ov-border); stroke-width: 8; }
.ring-value {
  fill: none;
  stroke: var(--ov-ok);
  stroke-width: 8;
  stroke-linecap: round;
  transition: stroke-dashoffset 0.6s ease, stroke 0.3s;
}
.tone-warn .ring-value { stroke: var(--ov-warn); }
.tone-danger .ring-value { stroke: var(--ov-danger); }

.ring-label {
  position: absolute;
  inset: 0;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
}
.ring-score { font-size: 34px; font-weight: 600; line-height: 1; font-variant-numeric: tabular-nums; }
.ring-status { margin-top: 4px; font-size: 12px; color: var(--ov-muted); }

.components { list-style: none; flex: 1 1 150px; min-width: 0; display: flex; flex-direction: column; gap: 9px; }
.components li { display: flex; align-items: center; justify-content: space-between; gap: 12px; font-size: 13px; }
.c-name { color: var(--ov-text); }
.c-status { font-size: 12px; font-weight: 500; }
.s-healthy { color: var(--ov-ok); }
.s-unhealthy { color: var(--ov-danger); }
.s-unknown { color: var(--ov-muted); }
</style>
