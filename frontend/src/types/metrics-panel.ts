/**
 * Universal Metrics Panel Types
 * Типы для универсальной компоненты отображения метрик
 */

/**
 * Конфигурация gauge (круговой) метрики
 */
export interface GaugeMetricConfig {
  /** Уникальный ID метрики */
  id: string

  /** Название метрики */
  title: string

  /** Единица измерения */
  unit?: 'bytes' | 'cores' | 'count' | 'percent'

  /** Подзаголовок метрики */
  subtitle?: string

  /** Функция получения текущего значения */
  getValue: () => number

  /** Функция получения процента использования (0-100) */
  getPercent: () => number

  /** Функция получения максимального значения */
  getTotal: () => number
}

/**
 * Конфигурация chart (график) метрики
 */
export interface ChartMetricConfig {
  /** Уникальный ID метрики */
  id: string

  /** Название графика */
  title: string

  /** Единица измерения */
  unit?: string

  /** Hex цвет графика (#RRGGBB) */
  color?: string

  /** Максимальное значение Y оси */
  maxY?: number

  /** Использовать многострочный график (MultiLineChart вместо MetricsChart) */
  multiLine?: boolean

  /** Функция загрузки данных для графика */
  fetchData: (startUnix: number, endUnix: number, step: number) => Promise<any>

  /** Функция форматирования значения для отображения */
  formatValue?: (value: number) => string
}

/**
 * Props для UniversalMetricsPanel компоненты
 */
export interface UniversalMetricsPanelProps {
  /** Имя кластера (обязательное) */
  clusterName: string

  /** Массив конфигураций gauge-метрик */
  gaugeMetrics?: GaugeMetricConfig[]

  /** Массив конфигураций chart-метрик */
  chartMetrics?: ChartMetricConfig[]

  /** Количество колонок в сетке gauge-метрик (1-4) */
  gaugeGridCols?: number

  /** Количество колонок в сетке chart-метрик (1-4) */
  chartGridCols?: number

  /** Интервал обновления метрик в ms */
  refreshIntervalMs?: number

  /** Сообщение когда metrics stack недоступен */
  notAvailableMessage?: string
}

/**
 * Обработанная gauge метрика (с вычисленными значениями)
 */
export interface ProcessedGaugeMetric extends GaugeMetricConfig {
  used: number
  percent: number
  total: number
}

/**
 * Обработанная chart метрика
 */
export interface ProcessedChartMetric extends ChartMetricConfig {
  color: string
}

/**
 * Стандартные форматировщики значений
 */
export const MetricsFormatters = {
  /**
   * Форматирование CPU (cores)
   * @example
   * formatCpu(0.5) // "500m"
   * formatCpu(1.5) // "1.50"
   */
  cpu: (v: number): string => {
    if (v < 0.001) return '0'
    if (v < 1) return (v * 1000).toFixed(0) + 'm'
    return v.toFixed(2)
  },

  /**
   * Форматирование Memory в MB
   * @example
   * formatMemoryMB(512) // "512.0 MB"
   * formatMemoryMB(1536) // "1.50 GB"
   */
  memoryMB: (v: number): string => {
    if (v < 1) return (v * 1024).toFixed(0) + ' KB'
    if (v >= 1024) return (v / 1024).toFixed(2) + ' GB'
    return v.toFixed(1) + ' MB'
  },

  /**
   * Форматирование Memory в bytes
   * @example
   * formatMemoryBytes(1024) // "1.0 KB"
   * formatMemoryBytes(1048576) // "1.0 MB"
   */
  memoryBytes: (v: number): string => {
    const units = ['B', 'KB', 'MB', 'GB', 'TB']
    if (v === 0) return '0 B'
    const i = Math.floor(Math.log(v) / Math.log(1024))
    const val = v / Math.pow(1024, i)
    return val.toFixed(1) + ' ' + units[i]
  },

  /**
   * Форматирование процентов
   * @example
   * formatPercent(50.5) // "50.5%"
   */
  percent: (v: number): string => {
    return v.toFixed(1) + '%'
  },

  /**
   * Форматирование Network (KB/s)
   * @example
   * formatNetwork(512) // "512.0 KB/s"
   * formatNetwork(2048) // "2.00 MB/s"
   */
  networkKBs: (v: number): string => {
    if (v >= 1024) return (v / 1024).toFixed(2) + ' MB/s'
    return v.toFixed(1) + ' KB/s'
  },

  /**
   * Форматирование Disk I/O (KB/s)
   * @example
   * formatDiskIO(1024) // "1.00 MB/s"
   */
  diskIOKBs: (v: number): string => {
    if (v >= 1024) return (v / 1024).toFixed(2) + ' MB/s'
    if (v >= 1) return v.toFixed(1) + ' KB/s'
    return (v * 1024).toFixed(0) + ' B/s'
  },

  /**
   * Форматирование количества операций
   * @example
   * formatOps(1234) // "1.2K"
   */
  ops: (v: number): string => {
    if (v >= 1000000) return (v / 1000000).toFixed(1) + 'M'
    if (v >= 1000) return (v / 1000).toFixed(1) + 'K'
    return v.toFixed(0)
  },

  /**
   * Форматирование с фиксированным количеством знаков
   */
  fixed: (decimals: number) => (v: number): string => v.toFixed(decimals),
}

/**
 * Рекомендуемые цвета для метрик
 */
export const MetricsColors = {
  cpu: '#3B82F6',          // Blue
  memory: '#8B5CF6',       // Purple
  disk: '#F59E0B',         // Amber
  network: '#10B981',      // Green
  networkRx: '#06B6D4',    // Cyan
  networkTx: '#10B981',    // Green
  requests: '#06B6D4',     // Cyan
  limits: '#EC4899',       // Pink
  usage: '#EF4444',        // Red
  available: '#6366F1',    // Indigo
  reads: '#3B82F6',        // Blue
  writes: '#F59E0B',       // Amber
} as const

/**
 * Стандартные конфигурации для разных типов ресурсов
 */

/**
 * Pod Metrics Presets
 */
export const PodMetricsPresets = {
  /**
   * CPU метрика для пода
   */
  cpu: (namespace: string, podName: string): ChartMetricConfig => ({
    id: 'pod-cpu',
    title: 'CPU Usage',
    unit: 'cores',
    color: MetricsColors.cpu,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.cpu,
  }),

  /**
   * Memory метрика для пода
   */
  memory: (namespace: string, podName: string): ChartMetricConfig => ({
    id: 'pod-memory',
    title: 'Memory Usage',
    unit: 'MB',
    color: MetricsColors.memory,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.memoryMB,
  }),

  /**
   * Network Receive метрика для пода
   */
  networkRx: (namespace: string, podName: string): ChartMetricConfig => ({
    id: 'pod-net-rx',
    title: 'Network Receive',
    unit: 'KB/s',
    color: MetricsColors.networkRx,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.networkKBs,
  }),

  /**
   * Network Transmit метрика для пода
   */
  networkTx: (namespace: string, podName: string): ChartMetricConfig => ({
    id: 'pod-net-tx',
    title: 'Network Transmit',
    unit: 'KB/s',
    color: MetricsColors.networkTx,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.networkKBs,
  }),
}

/**
 * Node Metrics Presets
 */
export const NodeMetricsPresets = {
  /**
   * CPU метрика для ноды
   */
  cpu: (nodeName: string): ChartMetricConfig => ({
    id: 'node-cpu',
    title: 'CPU Usage',
    unit: '%',
    color: MetricsColors.cpu,
    maxY: 100,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.percent,
  }),

  /**
   * Memory метрика для ноды
   */
  memory: (nodeName: string): ChartMetricConfig => ({
    id: 'node-memory',
    title: 'Memory Usage',
    unit: '%',
    color: MetricsColors.memory,
    maxY: 100,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.percent,
  }),

  /**
   * Disk метрика для ноды
   */
  disk: (nodeName: string): ChartMetricConfig => ({
    id: 'node-disk',
    title: 'Disk Usage',
    unit: '%',
    color: MetricsColors.disk,
    maxY: 100,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: MetricsFormatters.percent,
  }),

  /**
   * Network I/O метрика для ноды
   */
  networkIO: (nodeName: string): ChartMetricConfig => ({
    id: 'node-net-io',
    title: 'Network I/O',
    unit: 'MB/s',
    color: MetricsColors.network,
    fetchData: async () => [], // Переопределить в компоненте
    formatValue: (v: number) => v.toFixed(2),
  }),
}
