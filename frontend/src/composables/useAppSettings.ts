import { ref, readonly } from 'vue'
import { GetAppSettings, SaveAppSettings } from '../../wailsjs/go/main/App'
import { k8s } from '../../wailsjs/go/models'

const DEFAULT_INTERVAL = 10 // секунд

const settings = ref<k8s.KubronConfig>({
  lastActiveCluster: '',
  theme: 'dark',
  autoRefreshInterval: DEFAULT_INTERVAL,
})

const loaded = ref(false)

export function useAppSettings() {
  async function loadSettings() {
    try {
      const s = await GetAppSettings()
      settings.value = s
      // Если интервал не задан (0 или undefined), ставим дефолт
      if (!settings.value.autoRefreshInterval || settings.value.autoRefreshInterval < 0) {
        settings.value.autoRefreshInterval = DEFAULT_INTERVAL
      }
      loaded.value = true
    } catch (e) {
      console.error('Failed to load app settings:', e)
    }
  }

  async function saveSettings(newSettings: Partial<k8s.KubronConfig>) {
    settings.value = { ...settings.value, ...newSettings }
    try {
      await SaveAppSettings(settings.value)
    } catch (e) {
      console.error('Failed to save app settings:', e)
    }
  }

  function getRefreshIntervalMs(): number {
    const sec = settings.value.autoRefreshInterval || 0
    return sec > 0 ? sec * 1000 : 0
  }

  return {
    settings: readonly(settings),
    loaded: readonly(loaded),
    loadSettings,
    saveSettings,
    getRefreshIntervalMs,
  }
}
