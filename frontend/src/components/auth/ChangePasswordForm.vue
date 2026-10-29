<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <form class="auth-form" @submit.prevent="submit" autocomplete="off">
    <label class="auth-field">
      <span>{{ $t('auth.change.current') }}</span>
      <input ref="firstInput" v-model="oldPassword" type="password" autocomplete="current-password"
        :disabled="busy" required />
    </label>
    <label class="auth-field">
      <span>{{ $t('auth.change.new') }}</span>
      <input v-model="newPassword" type="password" autocomplete="new-password" :disabled="busy" required />
      <small>{{ $t('auth.fields.minLength', { min: MIN_PASSWORD }) }}</small>
    </label>
    <label class="auth-field">
      <span>{{ $t('auth.change.confirmNew') }}</span>
      <input v-model="confirm" type="password" autocomplete="new-password" :disabled="busy" required />
    </label>

    <p v-if="error" class="auth-error" role="alert">{{ error }}</p>

    <div class="auth-actions">
      <slot name="secondary" />
      <button class="auth-submit" type="submit" :disabled="busy || !filled">
        {{ busy ? $t('auth.change.submitting') : $t('auth.change.submit') }}
      </button>
    </div>
  </form>
</template>

<script lang="ts" setup>
import { computed, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { errorText } from '../../composables/errorText'
import { useAuth } from '../../composables/useAuth'
import { MIN_PASSWORD } from './constants'

const emit = defineEmits<{ (e: 'done'): void }>()

const { t } = useI18n()
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
    error.value = t('auth.errors.passwordTooShort', { min: MIN_PASSWORD })
    return
  }
  if (newPassword.value !== confirm.value) {
    error.value = t('auth.errors.passwordMismatch')
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


