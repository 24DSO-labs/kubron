<template>
  <section class="ov-card events">
    <h2 class="ov-card-title">Recent events</h2>

    <ul v-if="events.length" class="list">
      <li v-for="(ev, i) in events" :key="`${ev.time}/${ev.object}/${i}`" :title="`${ev.object}\n${ev.message}`">
        <span class="dot" :class="ev.type === 'Warning' ? 'dot-warn' : 'dot-ok'"></span>
        <span class="text">
          <span class="reason">{{ ev.reason }}</span>
          <span class="message">{{ ev.message }}</span>
        </span>
        <time class="time">{{ timeAgo(ev.time, now) }}</time>
      </li>
    </ul>
    <p v-else class="none">No events</p>

    <button type="button" class="more" @click="emit('viewAll')">View all events →</button>
  </section>
</template>

<script lang="ts" setup>
import { onMounted, onUnmounted, ref } from 'vue'
import { timeAgo } from './format'
import type { OverviewEvent } from '../../types/overview'

defineProps<{ events: OverviewEvent[] }>()
const emit = defineEmits<{ (e: 'viewAll'): void }>()

// «5m ago» должно идти вперёд и между обновлениями данных
const now = ref(Date.now())
let tick: ReturnType<typeof setInterval> | null = null
onMounted(() => { tick = setInterval(() => (now.value = Date.now()), 30_000) })
onUnmounted(() => { if (tick) clearInterval(tick) })
</script>

<style scoped>
.list { list-style: none; display: flex; flex-direction: column; gap: 11px; flex: 1; min-height: 0; overflow: hidden; }
.list li { display: flex; align-items: center; gap: 10px; font-size: 13px; min-width: 0; }
.dot { width: 7px; height: 7px; border-radius: 50%; flex-shrink: 0; }
.dot-ok { background: var(--ov-ok); }
.dot-warn { background: var(--ov-warn); }
.text { flex: 1; min-width: 0; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.reason { font-weight: 600; margin-right: 8px; }
.message { color: var(--ov-muted); }
.time { flex-shrink: 0; font-size: 12px; color: var(--ov-muted); }

.none { flex: 1; font-size: 13px; color: var(--ov-muted); }

.more {
  align-self: flex-start;
  margin-top: 12px;
  padding: 0;
  background: none;
  border: none;
  font: inherit;
  font-size: 12px;
  color: var(--ov-accent);
  cursor: pointer;
}
.more:hover { text-decoration: underline; }
</style>
