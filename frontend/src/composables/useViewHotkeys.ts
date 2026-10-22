import { onMounted, onUnmounted } from 'vue'

/** Alt+буква → переход к view (id — как в боковом меню). Работают, пока смонтирован вызывающий компонент */
const HOTKEYS: Record<string, string> = { p: 'pods', d: 'deployments', n: 'nodes', e: 'events', s: 'settings' }

function isTyping(target: EventTarget | null): boolean {
  const el = target as HTMLElement | null
  return !!el && (el.isContentEditable || ['INPUT', 'TEXTAREA', 'SELECT'].includes(el.tagName))
}

export function useViewHotkeys(navigate: (view: string) => void) {
  function onKeydown(e: KeyboardEvent) {
    if (!e.altKey || e.ctrlKey || e.metaKey || e.shiftKey || isTyping(e.target)) return
    const view = HOTKEYS[e.key.toLowerCase()]
    if (!view) return
    e.preventDefault()
    navigate(view)
  }
  onMounted(() => window.addEventListener('keydown', onKeydown))
  onUnmounted(() => window.removeEventListener('keydown', onKeydown))
}
