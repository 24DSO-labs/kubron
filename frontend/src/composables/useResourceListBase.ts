import { ref } from 'vue'
import type { Column, ContextMenuItem } from '../types' // Предполагаем, что типы в index.ts

export default function useResourceListBase(resourceType: string) {
  // Общие столбцы для всех ресурсов
  const defaultColumns: Column[] = [
    { key: 'name', label: 'Name' },
    { key: 'age', label: 'Age' },
    // Добавьте другие общие, если нужно (например, { key: 'age', label: 'Age' })
  ]

  // Общие пункты меню
  const defaultMenuItems: ContextMenuItem[] = [
    { label: 'Open Details', action: 'details' },
    { label: 'Logs', action: 'logs' },
    { label: 'Edit', action: 'yaml' },
    { label: 'Delete', action: 'delete', danger: true },
    // Добавьте 'Details' если нужно: { label: 'Open Details', action: 'details' }
  ]

  // Кастомизация по типу ресурса (если нужно расширять)
  let customColumns: Column[] = []
  let customMenuItems: ContextMenuItem[] = []

  if (resourceType === 'jobs') {
    customColumns = [
      { key: 'completions', label: 'Completions' },
      // Другие для jobs
    ]
    customMenuItems = [] // Пока пусто
  }

  // Слияние
  const mergedColumns = [...defaultColumns, ...customColumns]
  const mergedMenuItems = [...defaultMenuItems, ...customMenuItems]

  return {
    defaultColumns,
    defaultMenuItems,
    mergedColumns,
    mergedMenuItems
  }
}
