// Оформление кластеров: палитра, значки, метки окружения и чистые хелперы (без DOM и Wails).

export const MAX_CLUSTER_NAME = 60

export interface ClusterGlyph {
  /** Ключ, который хранится в метаданных кластера */
  key: string
  label: string
  /** Контуры в сетке 24×24: рисуются линией, поэтому наследуют цвет текста */
  paths: string[]
}

export const CLUSTER_GLYPHS: ClusterGlyph[] = [
  {
    key: 'server',
    label: 'Сервер',
    paths: [
      'M5 4h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2V6a2 2 0 0 1 2-2z',
      'M5 13h14a2 2 0 0 1 2 2v3a2 2 0 0 1-2 2H5a2 2 0 0 1-2-2v-3a2 2 0 0 1 2-2z',
      'M7 7.5h.01',
      'M7 16.5h.01',
    ],
  },
  {
    key: 'cloud',
    label: 'Облако',
    paths: ['M7 18a4 4 0 0 1-.6-7.96A5.5 5.5 0 0 1 17 8.5a4.75 4.75 0 0 1 0 9.5H7z'],
  },
  {
    key: 'database',
    label: 'Данные',
    paths: [
      'M4 5.5c0-1.66 3.58-3 8-3s8 1.34 8 3-3.58 3-8 3-8-1.34-8-3z',
      'M4 5.5v13c0 1.66 3.58 3 8 3s8-1.34 8-3v-13',
      'M4 12c0 1.66 3.58 3 8 3s8-1.34 8-3',
    ],
  },
  {
    key: 'shield',
    label: 'Защита',
    paths: ['M12 3l8 3v6c0 4.5-3.2 8-8 9-4.8-1-8-4.5-8-9V6l8-3z'],
  },
  {
    key: 'flask',
    label: 'Лаборатория',
    paths: ['M9 3h6', 'M10 3v6L5 18.5A2 2 0 0 0 6.8 21h10.4a2 2 0 0 0 1.8-2.5L14 9V3', 'M7.5 14h9'],
  },
  {
    key: 'cube',
    label: 'Куб',
    paths: ['M12 3l8 4.5v9L12 21l-8-4.5v-9L12 3z', 'M12 12l8-4.5', 'M12 12v9', 'M12 12L4 7.5'],
  },
]

/** Приглушённая палитра (оттенки одной яркости), чтобы цвета не спорили с интерфейсом */
export const CLUSTER_COLORS = [
  '#7aa2f7', // синий
  '#73daca', // бирюзовый
  '#9ece6a', // зелёный
  '#e0af68', // янтарный
  '#ff9e64', // оранжевый
  '#f7768e', // розовый
  '#bb9af7', // фиолетовый
  '#8b93a7', // графитовый
]

export const DEFAULT_CLUSTER_COLOR = CLUSTER_COLORS[0]

export interface ClusterEnvironment {
  /** '' — окружение не задано */
  key: string
  label: string
  color: string
}

export const CLUSTER_ENVIRONMENTS: ClusterEnvironment[] = [
  { key: '', label: 'Не задано', color: '#6c7086' },
  { key: 'dev', label: 'Dev', color: '#73daca' },
  { key: 'test', label: 'Test', color: '#e0af68' },
  { key: 'stage', label: 'Stage', color: '#ff9e64' },
  { key: 'prod', label: 'Prod', color: '#f7768e' },
]

/** Значок по ключу; для '' и устаревших эмодзи — null (рисуется монограмма) */
export function findGlyph(icon: string): ClusterGlyph | null {
  return CLUSTER_GLYPHS.find(g => g.key === icon) ?? null
}

/** Окружение по ключу; для '' и неизвестных ключей — null */
export function findEnvironment(key: string): ClusterEnvironment | null {
  if (!key) return null
  return CLUSTER_ENVIRONMENTS.find(e => e.key === key) ?? null
}

/**
 * Монограмма из названия: первые буквы двух первых слов, а у одного слова — его две первые буквы.
 * «nemo dev» → «ND», «production» → «PR», «k8s-prod-eu» → «KP».
 */
export function clusterInitials(name: string): string {
  const words = name.match(/[\p{L}\p{N}]+/gu) ?? []
  if (words.length === 0) return '?'
  const chars = (w: string) => Array.from(w)
  const letters = words.length >= 2
    ? chars(words[0])[0] + chars(words[1])[0]
    : chars(words[0]).slice(0, 2).join('')
  return letters.toUpperCase()
}

/** Цвет #rrggbb с прозрачностью alpha (0…1); значение другого формата возвращается как есть */
export function withAlpha(hex: string, alpha: number): string {
  if (!/^#[0-9a-fA-F]{6}$/.test(hex)) return hex
  const a = Math.round(Math.min(1, Math.max(0, alpha)) * 255)
  return hex + a.toString(16).padStart(2, '0')
}

/** Имя namespace по правилам Kubernetes (DNS-1123 label); пустое значение допустимо — «все» */
export function isValidNamespace(ns: string): boolean {
  if (ns === '') return true
  return ns.length <= 63 && /^[a-z0-9]([-a-z0-9]*[a-z0-9])?$/.test(ns)
}
