<template>
  <form class="auth-form" @submit.prevent="submit" autocomplete="off">
    <label class="auth-field">
      <span>Текущий пароль</span>
      <input ref="firstInput" v-model="oldPassword" type="password" autocomplete="current-password"
        :disabled="busy" required />
    </label>
    <label class="auth-field">
      <span>Новый пароль</span>
      <input v-model="newPassword" type="password" autocomplete="new-password" :disabled="busy" required />
      <small>Не короче {{ MIN_PASSWORD }} символов</small>
    </label>
    <label class="auth-field">
      <span>Повторите новый пароль</span>
      <input v-model="confirm" type="password" autocomplete="new-password" :disabled="busy" required />
    </label>

    <p v-if="error" class="auth-error" role="alert">{{ error }}</p>

    <div class="auth-actions">
      <slot name="secondary" />
      <button class="auth-submit" type="submit" :disabled="busy || !filled">
        {{ busy ? 'Сохранение…' : 'Сменить пароль' }}
      </button>
    </div>
  </form>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { errorText, useAuth } from '../../composables/useAuth'
import { MIN_PASSWORD } from './constants'

const emit = defineEmits<{ (e: 'done'): void }>()

const auth = useAuth()
const oldPassword = ref('')
const newPassword = ref('')
const confirm = ref('')
const busy = ref(false)
const error = ref('')
const firstInput = ref<HTMLInputElement | null>(null)

const filled = computed(() => !!oldPassword.value && !!newPassword.value && !!confirm.value)

onMounted(() => firstInput.value?.focus())

async function submit() {
  error.value = ''
  if (newPassword.value.length < MIN_PASSWORD) {
    error.value = `Пароль должен содержать не менее ${MIN_PASSWORD} символов`
    return
  }
  if (newPassword.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  busy.value = true
  try {
    await auth.changePassword(oldPassword.value, newPassword.value)
    emit('done')
  } catch (e) {
    error.value = errorText(e)
  } finally {
    busy.value = false
  }
}
</script>


