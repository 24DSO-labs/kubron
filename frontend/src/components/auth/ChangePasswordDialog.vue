<template>
  <div class="cp-overlay" @click.self="emit('close')" @keydown.esc="emit('close')">
    <div class="cp-dialog" role="dialog" aria-modal="true" aria-label="Смена пароля">
      <h3>Смена пароля</h3>
      <ChangePasswordForm @done="onDone">
        <template #secondary>
          <button class="auth-link" type="button" @click="emit('close')">Отмена</button>
        </template>
      </ChangePasswordForm>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { useNotifications } from '../../composables/useNotifications'
import ChangePasswordForm from './ChangePasswordForm.vue'

const emit = defineEmits<{ (e: 'close'): void }>()
const notifications = useNotifications()

function onDone() {
  notifications.notifySuccess('Пароль изменён')
  emit('close')
}
</script>

<style scoped>
.cp-overlay {
  position: fixed; inset: 0; z-index: 1000; display: flex; align-items: center; justify-content: center;
  background: rgba(0, 0, 0, 0.55);
}
.cp-dialog {
  width: 100%; max-width: 380px; padding: 24px; background: #181825;
  border: 1px solid #313244; border-radius: 14px; box-shadow: 0 24px 60px rgba(0, 0, 0, 0.5);
}
h3 { margin: 0 0 16px; color: #89b4fa; font-size: 17px; text-align: left; }
</style>

