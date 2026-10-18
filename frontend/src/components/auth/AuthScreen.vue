<template>
  <div class="auth-screen">
    <div class="auth-card">
      <img src="../../assets/images/logo.svg" alt="Kubron" class="auth-logo" />

      <!-- Загрузка -->
      <p v-if="mode === 'loading'" class="auth-hint">Загрузка…</p>

      <!-- Критическая ошибка хранилища пользователей: входить нельзя -->
      <template v-else-if="mode === 'fatal'">
        <h1>Вход недоступен</h1>
        <p class="auth-error" role="alert">{{ auth.fatalError.value }}</p>
        <p class="auth-hint">
          Файл пользователей <code>~/.kubron/users.json</code> не удалось прочитать.
          Восстановите его из резервной копии или обратитесь к администратору.
        </p>
      </template>

      <!-- Первый запуск -->
      <template v-else-if="mode === 'setup'">
        <h1>Добро пожаловать в Kubron</h1>
        <p class="auth-hint">Первый запуск: создайте учётную запись администратора.</p>
        <form class="auth-form" @submit.prevent="submitSetup" autocomplete="off">
          <label class="auth-field">
            <span>Логин</span>
            <input ref="firstInput" v-model="username" autocomplete="username" :disabled="busy" required />
            <small>3–32 символа: латиница, цифры, «.», «-», «_»</small>
          </label>
          <label class="auth-field">
            <span>Пароль</span>
            <input v-model="password" type="password" autocomplete="new-password" :disabled="busy" required />
            <small>Не короче {{ MIN_PASSWORD }} символов</small>
          </label>
          <label class="auth-field">
            <span>Повторите пароль</span>
            <input v-model="confirm" type="password" autocomplete="new-password" :disabled="busy" required />
          </label>
          <p v-if="error" class="auth-error" role="alert">{{ error }}</p>
          <div class="auth-actions">
            <button class="auth-submit" type="submit" :disabled="busy || !username || !password || !confirm">
              {{ busy ? 'Создание…' : 'Создать администратора' }}
            </button>
          </div>
        </form>
      </template>

      <!-- Вход -->
      <template v-else-if="mode === 'login'">
        <h1>Вход</h1>
        <form class="auth-form" @submit.prevent="submitLogin">
          <label class="auth-field">
            <span>Логин</span>
            <input ref="firstInput" v-model="username" autocomplete="username" :disabled="busy" required />
          </label>
          <label class="auth-field">
            <span>Пароль</span>
            <input v-model="password" type="password" autocomplete="current-password" :disabled="busy" required />
          </label>
          <p v-if="error" class="auth-error" role="alert">{{ error }}</p>
          <div class="auth-actions">
            <button class="auth-submit" type="submit" :disabled="busy || !username || !password">
              {{ busy ? 'Вход…' : 'Войти' }}
            </button>
          </div>
        </form>
      </template>

      <!-- Обязательная смена временного пароля -->
      <template v-else-if="mode === 'change-password'">
        <h1>Смена пароля</h1>
        <p class="auth-hint">
          Вам выдан временный пароль. Задайте собственный, чтобы продолжить работу.
        </p>
        <ChangePasswordForm>
          <template #secondary>
            <button class="auth-link" type="button" @click="auth.logout()">Выйти</button>
          </template>
        </ChangePasswordForm>
      </template>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { nextTick, ref, watch } from 'vue'
import { errorText, useAuth } from '../../composables/useAuth'
import ChangePasswordForm from './ChangePasswordForm.vue'
import { MIN_PASSWORD } from './constants'

const props = defineProps<{ mode: 'loading' | 'fatal' | 'setup' | 'login' | 'change-password' }>()

const auth = useAuth()
const username = ref('')
const password = ref('')
const confirm = ref('')
const busy = ref(false)
const error = ref('')
const firstInput = ref<HTMLInputElement | null>(null)

// Ошибки и поля не переходят между режимами; фокус — на первое поле
watch(
  () => props.mode,
  async () => {
    error.value = ''
    password.value = ''
    confirm.value = ''
    await nextTick()
    firstInput.value?.focus()
  },
  { immediate: true },
)

async function submitSetup() {
  error.value = ''
  if (password.value.length < MIN_PASSWORD) {
    error.value = `Пароль должен содержать не менее ${MIN_PASSWORD} символов`
    return
  }
  if (password.value !== confirm.value) {
    error.value = 'Пароли не совпадают'
    return
  }
  await run(() => auth.setup(username.value, password.value))
}

async function submitLogin() {
  error.value = ''
  await run(() => auth.login(username.value, password.value))
}

async function run(fn: () => Promise<void>) {
  busy.value = true
  try {
    await fn()
  } catch (e) {
    error.value = errorText(e)
    password.value = ''
    confirm.value = ''
  } finally {
    busy.value = false
  }
}
</script>

<style src="./auth.css"></style>
<style scoped>
.auth-screen {
  position: fixed;
  inset: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background:
    radial-gradient(900px 500px at 15% -10%, rgba(137, 180, 250, 0.14), transparent 60%),
    radial-gradient(700px 420px at 100% 110%, rgba(203, 166, 247, 0.10), transparent 60%),
    #11111b;
  overflow: auto;
}
.auth-card {
  width: 100%;
  max-width: 400px;
  padding: 32px 32px 28px;
  background: #181825;
  border: 1px solid #313244;
  border-radius: 16px;
  box-shadow: 0 24px 60px rgba(0, 0, 0, 0.45);
  text-align: center;
}
.auth-logo { height: 36px; margin-bottom: 18px; }
h1 { margin: 0 0 8px; font-size: 20px; font-weight: 700; color: #cdd6f4; }
.auth-hint { margin: 0 0 18px; font-size: 13px; line-height: 1.5; color: #a6adc8; }
.auth-hint code { background: #11111b; padding: 1px 5px; border-radius: 4px; color: #cdd6f4; }
.auth-card > .auth-error { text-align: left; margin-bottom: 14px; }
</style>
