import { ref } from 'vue'

export const useLogsPanel = () => {
  const isOpen = ref(false)
  const logsData = ref<{ resource: any; selectedContainer: string; containers: any[] } | null>(null)

  const openLogs = (data: { resource: any; selectedContainer: string; containers: any[] }) => {
    logsData.value = data
    isOpen.value = true
  }

  const closeLogs = () => {
    isOpen.value = false
    logsData.value = null
  }

  return { isOpen, logsData, openLogs, closeLogs }
}
