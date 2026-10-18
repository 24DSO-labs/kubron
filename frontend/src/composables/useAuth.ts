import { computed, readonly, ref } from 'vue'
import { callApp } from '../resources/wails'
import type { Permission, UserInfo } from '../types/auth'

/**
 * Состояние аутентификации — одно на всё приложение (модульный синглтон).
 *
 * Важно: это только удобство интерфейса. Права на самом деле проверяет Go-бэкенд
 * (internal/auth) в каждом методе; скрытая кнопка безопасность не обеспечивает.
 */

/** Экран, который сейчас должен показывать приложение */
export type AuthScreen = 'loading' | 'fatal' | 'setup' | 'login' | 'change-password' | 'app'

const initialized = ref(false)
const needsSetup = ref(false)
const user = ref<UserInfo | null>(null)
const fatalError = ref('')

const screen = computed<AuthScreen>(() => {
  if (fatalError.value) return 'fatal'
  if (!initialized.value) return 'loading'
  if (needsSetup.value) return 'setup'
  if (!user.value) return 'login'
  if (user.value.mustChangePassword) return 'change-password'
  return 'app'
})

/** Текст ошибки из Wails (строка или Error) */
export function errorText(e: unknown): string {
  if (typeof e === 'string') return e
  const msg = (e as { message?: string } | null)?.message
  return msg || String(e)
}

/** Права вошедшего пользователя; пока никто не вошёл — ничего не разрешено */
export function can(permission: Permission): boolean {
  const u = user.value
  return !!u && !u.mustChangePassword && u.permissions.includes(permission)
}

export function useAuth() {
  async function init() {
    try {
      const st = await callApp('AuthStatus')
      needsSetup.value = !!st.needsSetup
      user.value = (st.user as UserInfo | null) ?? null
      fatalError.value = ''
    } catch (e) {
      fatalError.value = errorText(e)
    } finally {
      initialized.value = true
    }
  }

  async function setup(username: string, password: string) {
    user.value = (await callApp('SetupAdmin', username.trim(), password)) as UserInfo
    needsSetup.value = false
  }

  async function login(username: string, password: string) {
    user.value = (await callApp('Login', username.trim(), password)) as UserInfo
  }

  async function logout() {
    try {
      await callApp('Logout')
    } finally {
      // Перезагрузка сбрасывает всё состояние интерфейса (кластеры, вкладки, кэши списков):
      // следующий пользователь не увидит данных предыдущего.
      window.location.reload()
    }
  }

  async function changePassword(oldPassword: string, newPassword: string) {
    await callApp('ChangePassword', oldPassword, newPassword)
    user.value = (await callApp('CurrentUser')) as UserInfo
  }

  return {
    screen: readonly(screen),
    user: readonly(user),
    fatalError: readonly(fatalError),
    init,
    setup,
    login,
    logout,
    changePassword,
    can,
  }
}
