# Руководство по использованию базовых компонентов K8s

## Содержание
1. [K8sResourceListBase - для списков ресурсов](#k8sresourcelistbase)
2. [K8sResourceDetailBase - для деталей ресурсов](#k8sresourcedetailbase)
3. [Примеры реализации](#примеры)

---

## K8sResourceListBase

Базовый компонент для отображения списков K8s ресурсов с фильтрацией, поиском, сортировкой и двухуровневым контекстным меню.

### Props

```typescript
{
  items: any[]                    // Массив ресурсов для отображения
  columns: Array<{                // Конфигурация колонок таблицы
    key: string                   // Ключ свойства ресурса
    label: string                 // Отображаемое имя колонки
    sortable?: boolean            // Можно ли сортировать (по умолчанию true)
    width?: string                // Ширина колонки (CSS)
  }>
  resourceType: string            // Тип ресурса (напр. 'Pod', 'Deployment')
  resourceIcon?: string           // Иконка для ресурса (emoji)
  enableSecondLevel?: boolean     // Включить второй уровень меню (по умолчанию false)
  showNamespace?: boolean         // Показывать колонку namespace (по умолчанию true)
  filterableFields?: string[]     // Поля для фильтрации (напр. ['name', 'namespace', 'status'])
  pageSize?: number               // Размер страницы для пагинации (по умолчанию 50)
}
```

### Emits (события)

```typescript
{
  'row-click'(row: any): void           // Клик по строке - открывает детали
  'context-menu'(action: {              // Контекстное меню - первый уровень
    action: string
    row: any
  }): void
  'context-submenu'(action: {           // Контекстное меню - второй уровень
    action: string
    subaction: string
    row: any
  }): void
  'refresh'(): void                     // Кнопка обновления
}
```

### Slots

Компонент НЕ имеет slots для контента. Он сам рендерит таблицу на основе `items` и `columns`.

### Использование

```vue
<template>
  <K8sResourceListBase
    :items="pods"
    :columns="podColumns"
    resource-type="Pod"
    resource-icon="📦"
    :enable-second-level="true"
    :filterable-fields="['name', 'namespace', 'status']"
    @row-click="openPodDetails"
    @context-menu="handlePodAction"
    @context-submenu="handlePodSubAction"
    @refresh="refreshPods"
  />
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import K8sResourceListBase from '@/components/views/K8sResourceListBase.vue'

const pods = ref([
  { name: 'pod-1', namespace: 'default', status: 'Running', age: '2d', ... },
  { name: 'pod-2', namespace: 'kube-system', status: 'Running', age: '5d', ... },
])

const podColumns = [
  { key: 'name', label: 'Name', sortable: true },
  { key: 'namespace', label: 'Namespace', sortable: true },
  { key: 'status', label: 'Status', sortable: true },
  { key: 'age', label: 'Age', sortable: true },
]

function openPodDetails(row) {
  // Открыть панель деталей пода
  console.log('Open details for:', row.name)
}

function handlePodAction(action) {
  // Обработка первого уровня меню (Attach, Terminal, Logs, etc)
  console.log('Action:', action.action, 'Pod:', action.row.name)
}

function handlePodSubAction(action) {
  // Обработка второго уровня меню (выбор контейнера для Attach, Terminal)
  console.log('SubAction:', action.action, action.subaction, 'Pod:', action.row.name)
}

function refreshPods() {
  // Перез загрузить список подов
  console.log('Refresh pods')
}
</script>
```

---

## K8sResourceDetailBase

Базовый компонент для отображения деталей одного K8s ресурса с вкладками и действиями.

### Props

```typescript
{
  resource: any                   // Объект ресурса с данными
  resourceIcon?: string           // Иконка ресурса (emoji)
  showMetaInfo?: boolean          // Показать базовую информацию (Name, Namespace, Age) - по умолчанию true
  customTabs?: Array<{            // Дополнительные вкладки
    id: string                    // Уникальный ID вкладки
    label: string                 // Отображаемое имя вкладки
  }>
}
```

### Emits (события)

```typescript
{
  'close'(): void                        // Кнопка закрытия панели
  'action'(actionId: string, data?: any): void  // Пользовательские события
}
```

### Slots

```vue
<K8sResourceDetailBase :resource="resource">
  <!-- Слот для кнопок действий в header -->
  <template #actions>
    <button @click="doAction">Action</button>
  </template>

  <!-- Слот для содержимого вкладки Properties -->
  <template #properties>
    <div>Основные свойства ресурса</div>
  </template>

  <!-- Слот для содержимого вкладки Events -->
  <template #events>
    <div>События ресурса</div>
  </template>

  <!-- Слот для дополнительных вкладок (customTabs) -->
  <template #tab-containers>
    <div>Содержимое вкладки Containers</div>
  </template>

  <template #tab-logs>
    <div>Содержимое вкладки Logs</div>
  </template>
</K8sResourceDetailBase>
```

### Использование

```vue
<template>
  <K8sResourceDetailBase
    :resource="pod"
    resource-icon="📦"
    :custom-tabs="customTabs"
    @close="closePodDetails"
  >
    <!-- Действия -->
    <template #actions>
      <button @click="attachToPod">Attach</button>
      <button @click="openTerminal">Terminal</button>
      <button @click="viewLogs">Logs</button>
      <button @click="deletePod" class="btn-danger">Delete</button>
    </template>

    <!-- Основные свойства -->
    <template #properties>
      <div class="pod-info">
        <div class="row">
          <span class="label">Status:</span>
          <span>{{ pod.status }}</span>
        </div>
        <div class="row">
          <span class="label">Node:</span>
          <span>{{ pod.node }}</span>
        </div>
        <div class="row">
          <span class="label">IP:</span>
          <span>{{ pod.ip }}</span>
        </div>
      </div>
    </template>

    <!-- События -->
    <template #events>
      <div v-if="pod.events?.length" class="events-list">
        <div v-for="event in pod.events" :key="event.id" class="event-item">
          {{ event.message }}
        </div>
      </div>
      <div v-else class="empty">No events</div>
    </template>

    <!-- Вкладка Containers (дополнительная) -->
    <template #tab-containers>
      <div v-for="container in pod.containers" :key="container.name">
        <h4>{{ container.name }}</h4>
        <p>Image: {{ container.image }}</p>
      </div>
    </template>
  </K8sResourceDetailBase>
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import K8sResourceDetailBase from '@/components/panels/K8sResourceDetailBase.vue'

const pod = ref({
  name: 'my-pod',
  namespace: 'default',
  age: '2d',
  status: 'Running',
  node: 'node-1',
  ip: '10.0.0.1',
  events: [
    { id: '1', message: 'Pod created' },
    { id: '2', message: 'Container started' },
  ],
  containers: [
    { name: 'web', image: 'nginx:latest' },
  ],
})

const customTabs = [
  { id: 'containers', label: 'Containers' },
]

function closePodDetails() {
  console.log('Close details')
}

function attachToPod() {
  console.log('Attach to pod')
}

function openTerminal() {
  console.log('Open terminal')
}

function viewLogs() {
  console.log('View logs')
}

function deletePod() {
  console.log('Delete pod')
}
</script>
```

---

## Примеры

### Пример 1: Простой список Pods

```vue
<!-- PodsView.vue -->
<template>
  <K8sResourceListBase
    :items="pods"
    :columns="podColumns"
    resource-type="Pod"
    resource-icon="📦"
    :enable-second-level="true"
    :filterable-fields="['name', 'namespace', 'status']"
    @row-click="onPodSelected"
    @context-menu="handlePodContextMenu"
    @context-submenu="handlePodSubAction"
  />

  <!-- Панель деталей пода -->
  <PodDetailPanel
    v-if="selectedPod"
    :pod="selectedPod"
    @close="selectedPod = null"
    @delete="deletePod"
    @edit="editPod"
    @attach="attachContainer"
    @terminal="openTerminal"
    @logs="viewLogs"
    @evict="evictPod"
  />
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import K8sResourceListBase from '@/components/views/K8sResourceListBase.vue'
import PodDetailPanel from '@/components/panels/k8s_resource/PodDetailPanel.vue'

const pods = ref([])
const selectedPod = ref(null)

const podColumns = [
  { key: 'name', label: 'Name' },
  { key: 'namespace', label: 'Namespace' },
  { key: 'status', label: 'Status' },
  { key: 'restarts', label: 'Restarts' },
  { key: 'age', label: 'Age' },
]

function onPodSelected(pod) {
  selectedPod.value = pod
}

function handlePodContextMenu(action) {
  const { action: actionType, row } = action
  console.log(`${actionType} for pod ${row.name}`)
}

function handlePodSubAction(action) {
  const { action: mainAction, subaction, row } = action
  console.log(`${mainAction} > ${subaction} for pod ${row.name}`)
}
</script>
```

### Пример 2: Список Deployments

```vue
<!-- DeploymentsView.vue -->
<template>
  <K8sResourceListBase
    :items="deployments"
    :columns="deploymentColumns"
    resource-type="Deployment"
    resource-icon="📦"
    :enable-second-level="false"
    :filterable-fields="['name', 'namespace', 'status']"
    @row-click="onDeploymentSelected"
    @context-menu="handleDeploymentAction"
  />

  <DeploymentDetailPanel
    v-if="selectedDeployment"
    :deployment="selectedDeployment"
    @close="selectedDeployment = null"
  />
</template>

<script lang="ts" setup>
import { ref } from 'vue'
import K8sResourceListBase from '@/components/views/K8sResourceListBase.vue'
import DeploymentDetailPanel from '@/components/panels/k8s_resource/DeploymentDetailPanel.vue'

const deployments = ref([])
const selectedDeployment = ref(null)

const deploymentColumns = [
  { key: 'name', label: 'Name' },
  { key: 'namespace', label: 'Namespace' },
  { key: 'replicas', label: 'Replicas' },
  { key: 'status', label: 'Status' },
  { key: 'age', label: 'Age' },
]

function onDeploymentSelected(deployment) {
  selectedDeployment.value = deployment
}

function handleDeploymentAction(action) {
  console.log('Action:', action)
}
</script>
```

---

## Чек-лист при добавлении новой View

При создании новой View для нового типа ресурса:

1. ✅ Создать файл `views/k8s_resource/MyResourceView.vue`
2. ✅ Импортировать `K8sResourceListBase`
3. ✅ Определить `columns` массив с нужными полями
4. ✅ Передать `items`, `columns`, `resource-type`, `resource-icon`
5. ✅ Обработать события `@row-click`, `@context-menu` (и `@context-submenu` если нужен второй уровень)
6. ✅ Создать соответствующий DetailPanel в `panels/k8s_resource/MyResourceDetailPanel.vue`
7. ✅ Показывать DetailPanel при выборе строки
8. ✅ Передавать нужные слоты (#properties, #events, #actions, #tab-xxx)

## Чек-лист при добавлении новой DetailPanel

При создании новой DetailPanel для нового типа ресурса:

1. ✅ Создать файл `panels/k8s_resource/MyResourceDetailPanel.vue`
2. ✅ Импортировать `K8sResourceDetailBase`
3. ✅ Определить props для данных ресурса
4. ✅ Определить emits для событий (close, delete, edit, etc)
5. ✅ Использовать slot `#actions` для кнопок
6. ✅ Использовать slot `#properties` для основной информации
7. ✅ Использовать slot `#events` для событий ресурса
8. ✅ Добавить дополнительные вкладки через `#tab-xxx` слоты если нужны
9. ✅ Передать `custom-tabs` prop если добавлены дополнительные вкладки

---

## Важные замечания

- **K8sResourceListBase** автоматически обрабатывает фильтрацию, сортировку и пагинацию
- **K8sResourceDetailBase** предоставляет систему вкладок - Properties и Events всегда присутствуют
- Второй уровень контекстного меню (второй клик по меню) активируется через `enable-second-level`
- Все детальные панели должны использовать `K8sResourceDetailBase` как обёртку
- События ресурса автоматически добавляются как вкладка Events
