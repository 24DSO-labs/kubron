<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="notifications-container">
    <div
      v-for="n in notifications"
      :key="n.id"
      class="notification-toast"
      :class="`notification-${n.type}`"
      @click="$emit('dismiss', n.id)"
    >
      <span class="notification-icon">
        {{ n.type === 'error' ? '❌' : n.type === 'success' ? '✅' : 'ℹ️' }}
      </span>
      {{ n.message }}
    </div>
  </div>
</template>

<script lang="ts" setup>
import type { Notification } from '../composables/useNotifications'

defineProps<{
  notifications: Notification[]
}>()

defineEmits<{
  dismiss: [id: number]
}>()
</script>

<style scoped>
.notifications-container {
  position: fixed;
  top: 16px;
  right: 16px;
  z-index: 10000;
  display: flex;
  flex-direction: column;
  gap: 8px;
  max-width: 420px;
}

.notification-toast {
  padding: 12px 16px;
  border-radius: 8px;
  cursor: pointer;
  color: #fff;
  font-size: 13px;
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.3);
  animation: slideIn 0.2s ease-out;
}

.notification-error {
  background: #e53e3e;
}

.notification-success {
  background: #38a169;
}

.notification-info {
  background: #3182ce;
}

.notification-icon {
  margin-right: 8px;
}

@keyframes slideIn {
  from {
    opacity: 0;
    transform: translateX(40px);
  }
  to {
    opacity: 1;
    transform: translateX(0);
  }
}
</style>