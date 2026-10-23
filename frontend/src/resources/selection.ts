import { computed, shallowRef, watch } from 'vue'

/** Копия набора с переключённым ключом (набор не мутируем: ref следит за заменой значения) */
export function toggledKey(set: ReadonlySet<string>, key: string): Set<string> {
  const next = new Set(set)
  if (!next.delete(key)) next.add(key)
  return next
}

/** Оставляет в выборе только ключи, которые ещё есть в списке; тот же объект, если ничего не изменилось */
export function prunedKeys(set: ReadonlySet<string>, alive: Iterable<string>): ReadonlySet<string> {
  const live = new Set(alive)
  const kept = [...set].filter(k => live.has(k))
  return kept.length === set.size ? set : new Set(kept)
}

/**
 * Выбор строк списка (чекбоксы, как в Lens). Ключ строки — тот же, что у таблицы (ns/name).
 * Выбор живёт только среди видимых строк: после поиска или обновления списка пропавшие строки
 * из выбора выпадают, поэтому «−» никогда не удалит то, чего пользователь не видит.
 */
export function useSelection<T>(items: () => readonly T[], keyOf: (item: T) => string) {
  const selected = shallowRef<ReadonlySet<string>>(new Set())

  const selectedItems = computed(() => items().filter(i => selected.value.has(keyOf(i))))
  const count = computed(() => selectedItems.value.length)
  const allSelected = computed(() => items().length > 0 && count.value === items().length)
  const someSelected = computed(() => count.value > 0 && !allSelected.value)

  const isSelected = (item: T) => selected.value.has(keyOf(item))

  function toggle(item: T) {
    selected.value = toggledKey(selected.value, keyOf(item))
  }

  /** Все видимые строки; если выбраны все — снять выбор */
  function toggleAll() {
    selected.value = allSelected.value ? new Set() : new Set(items().map(keyOf))
  }

  function clear() {
    if (selected.value.size) selected.value = new Set()
  }

  // Новый срез списка: оставляем только те ключи, что в нём ещё есть
  watch(items, list => {
    if (!selected.value.size) return
    selected.value = prunedKeys(selected.value, list.map(keyOf))
  })

  return { selected, selectedItems, count, allSelected, someSelected, isSelected, toggle, toggleAll, clear }
}
