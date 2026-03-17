<template>
  <div>
    <ContentHeader icon="⚡" title="Events" :count="items.length" @refresh="$emit('refresh')" />
    <table v-if="items.length>0" class="table">
      <thead><tr><th>Type</th><th>Reason</th><th>Object</th><th>Message</th><th>Count</th><th>Age</th><th>Source</th></tr></thead>
      <tbody>
        <tr v-for="(ev,i) in items" :key="i" :class="{'event-row-warning':ev.type==='Warning'}">
          <td><span class="event-type" :class="'event-'+ev.type.toLowerCase()">{{ev.type}}</span></td>
          <td class="event-reason">{{ev.reason}}</td>
          <td class="resource-name">{{ev.object}}</td>
          <td class="event-message">{{ev.message}}</td>
          <td>{{ev.count}}</td>
          <td>{{ev.age}}</td>
          <td class="ns-label">{{ev.source}}</td>
        </tr>
      </tbody>
    </table>
    <div v-else class="empty">Нет событий</div>
  </div>
</template>

<script lang="ts" setup>
import type { EventInfo } from '../../types'
import ContentHeader from '../ContentHeader.vue'
defineProps<{ items: EventInfo[] }>()
defineEmits<{ (e: 'refresh'): void }>()
</script>