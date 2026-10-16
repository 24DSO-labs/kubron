import { defineAsyncComponent, type Component } from 'vue'

export interface ViewConfig {
  component: Component
  dataKey?: string // ключ для items
  hasSearch?: boolean
  extraEvents?: string[]
  extraProps?: Record<string, any>
}

// Ленивая загрузка для тяжёлых/редких компонентов
const lazy = (path: string) => defineAsyncComponent(() => import(`../components/views/${path}.vue`))
const lazyK8s = (path: string) => defineAsyncComponent(() => import(`../components/views/k8s_resource/${path}.vue`))

export const viewRegistry: Record<string, ViewConfig> = {
  overview: {
    component: lazy('OverviewView'),
  },
  cronjobs: {
    component: lazyK8s('CronJobsView'),
    dataKey: 'filteredCronJobs',
    hasSearch: true,
  },
  ingresses: {
    component: lazyK8s('IngressesView'),
    dataKey: 'filteredIngresses',
    hasSearch: true,
  },
  endpoints: {
    component: lazyK8s('EndpointsView'),
    dataKey: 'filteredEndpoints',
    hasSearch: true,
  },
  ingressclasses: {
    component: lazyK8s('IngressClassesView'),
    dataKey: 'filteredIngressClasses',
    hasSearch: true,
  },
  'network-policies': {
    component: lazyK8s('NetworkPoliciesView'),
    dataKey: 'filteredNetworkPolicies',
    hasSearch: true,
  },
  portforwarding: {
    component: lazy('PortForwardingView'),
    dataKey: 'portForwards',
  },
  pv: {
    component: lazyK8s('PVView'),
    dataKey: 'pvs',
  },
  pvc: {
    component: lazyK8s('PVCView'),
    dataKey: 'filteredPVCs',
    hasSearch: true,
  },
  storageclasses: {
    component: lazyK8s('StorageClassesView'),
    dataKey: 'filteredStorageClasses',
    hasSearch: true,
  },
  // namespaces: {
  //   component: lazyK8s('NamespacesView'),
  //   dataKey: 'namespaces',
  //   hasSearch: true,
  // },
  nodes: {
    component: lazy('NodesView'),
    dataKey: 'nodes',
  },
  events: {
    component: lazy('EventsView'),
    dataKey: 'events',
  },
  // configmaps: {
  //   component: lazyK8s('ConfigMapsView'),
  //   dataKey: 'filteredConfigMaps',
  //   hasSearch: true,
  // },
  resourcequotas: {
    component: lazyK8s('ResourceQuotasView'),
    dataKey: 'filteredResourceQuotas',
    hasSearch: true,
  },
  limitranges: {
    component: lazyK8s('LimitRangesView'),
    dataKey: 'filteredLimitRanges',
    hasSearch: true,
  },
  hpa: {
    component: lazyK8s('HPAView'),
    dataKey: 'filteredHPA',
    hasSearch: true,
  },
  pdb: {
    component: lazyK8s('PDBView'),
    dataKey: 'filteredPDB',
    hasSearch: true,
  },
  priorityclasses: {
    component: lazyK8s('PriorityClassesView'),
    dataKey: 'filteredPriorityClasses',
    hasSearch: true,
  },
  runtimeclasses: {
    component: lazyK8s('RuntimeClassesView'),
    dataKey: 'filteredRuntimeClasses',
    hasSearch: true,
  },
  leases: {
    component: lazyK8s('LeasesView'),
    dataKey: 'filteredLeases',
    hasSearch: true,
  },
  // serviceaccounts: {
  //   component: lazyK8s('ServiceAccountsView'),
  //   dataKey: 'filteredServiceAccounts',
  //   hasSearch: true,
  // },
  // roles: {
  //   component: lazyK8s('RolesView'),
  //   dataKey: 'filteredRoles',
  //   hasSearch: true,
  // },
  // rolebindings: {
  //   component: lazyK8s('RoleBindingsView'),
  //   dataKey: 'filteredRoleBindings',
  //   hasSearch: true,
  // },
  // clusterroles: {
  //   component: lazyK8s('ClusterRolesView'),
  //   dataKey: 'filteredClusterRoles',
  //   hasSearch: true,
  // },
  // clusterrolebindings: {
  //   component: lazyK8s('ClusterRoleBindingsView'),
  //   dataKey: 'filteredClusterRoleBindings',
  //   hasSearch: true,
  // },
  settings: {
    component: lazy('ClusterSettings'),
  },
}
