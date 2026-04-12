import { ref } from 'vue'
import type { Column, ContextMenuItem } from '../types' // Предполагаем, что типы в index.ts

export default function useResourceListBase(resourceType: string) {
  // Общие столбцы для всех ресурсов
  const defaultColumns: Column[] = [
    { key: 'name', label: 'Name' },
    { key: 'age', label: 'Age' },
    
  ]

  // Базовые пункты меню - для всех ресурсов
  const defaultMenuItems: ContextMenuItem[] = [
    { label: 'Open Details', action: 'details' },
    { label: 'Logs', action: 'logs' },
    { label: 'Edit', action: 'yaml' },
    { label: 'Delete', action: 'delete', danger: true },
  ]

  // Добавляем специфичные пункты для подов
  if (resourceType === 'pods') {
    // Вставляем Shell и Attach после Open Details
    defaultMenuItems.splice(1, 0,
      { label: 'Shell', action: 'shell' },
      { label: 'Attach', action: 'attach' }
    )
  }

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
