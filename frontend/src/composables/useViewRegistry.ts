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

export const viewRegistry: Record<string, ViewConfig> = {
  overview: {
    component: lazy('OverviewView'),
  },
  pods: {
    component: lazy('PodsView'),
    dataKey: 'filteredPods',
    hasSearch: true,
  },
  deployments: {
    component: lazy('DeploymentsView'),
    dataKey: 'filteredDeployments',
    hasSearch: true,
  },
  statefulsets: {
    component: lazy('StatefulSetsView'),
    dataKey: 'filteredStatefulSets',
    hasSearch: true,
  },
  daemonsets: {
    component: lazy('DaemonSetsView'),
    dataKey: 'filteredDaemonSets',
    hasSearch: true,
  },
  jobs: {
    component: lazy('JobsView'),
    dataKey: 'filteredJobs',
    hasSearch: true,
  },
  cronjobs: {
    component: lazy('CronJobsView'),
    dataKey: 'filteredCronJobs',
    hasSearch: true,
  },
  replicasets: {
    component: lazy('ReplicaSetsView'),
    dataKey: 'filteredReplicaSets',
    hasSearch: true,
  },
  services: {
    component: lazy('ServicesView'),
    dataKey: 'filteredServices',
    hasSearch: true,
  },
  ingresses: {
    component: lazy('IngressesView'),
    dataKey: 'filteredIngresses',
    hasSearch: true,
  },
  endpoints: {
    component: lazy('EndpointsView'),
    dataKey: 'filteredEndpoints',
    hasSearch: true,
  },
  ingressclasses: {
    component: lazy('IngressClassesView'),
    dataKey: 'filteredIngressClasses',
    hasSearch: true,
  },
  'network-policies': {
    component: lazy('NetworkPoliciesView'),
    dataKey: 'filteredNetworkPolicies',
    hasSearch: true,
  },
  portforwarding: {
    component: lazy('PortForwardingView'),
    dataKey: 'portForwards',
  },
  pv: {
    component: lazy('PVView'),
    dataKey: 'pvs',
  },
  pvc: {
    component: lazy('PVCView'),
    dataKey: 'filteredPVCs',
    hasSearch: true,
  },
  storageclasses: {
    component: lazy('StorageClassesView'),
    dataKey: 'filteredStorageClasses',
    hasSearch: true,
  },
  namespaces: {
    component: lazy('NamespacesView'),
    dataKey: 'namespaces',
    hasSearch: true,
  },
  nodes: {
    component: lazy('NodesView'),
    dataKey: 'nodes',
  },
  events: {
    component: lazy('EventsView'),
    dataKey: 'events',
  },
  configmaps: {
    component: lazy('ConfigMapsView'),
    dataKey: 'filteredConfigMaps',
    hasSearch: true,
  },
  secrets: {
    component: lazy('SecretsView'),
    dataKey: 'filteredSecrets',
    hasSearch: true,
  },
  resourcequotas: {
    component: lazy('ResourceQuotasView'),
    dataKey: 'filteredResourceQuotas',
    hasSearch: true,
  },
  limitranges: {
    component: lazy('LimitRangesView'),
    dataKey: 'filteredLimitRanges',
    hasSearch: true,
  },
  hpa: {
    component: lazy('HPAView'),
    dataKey: 'filteredHPA',
    hasSearch: true,
  },
  pdb: {
    component: lazy('PDBView'),
    dataKey: 'filteredPDB',
    hasSearch: true,
  },
  priorityclasses: {
    component: lazy('PriorityClassesView'),
    dataKey: 'filteredPriorityClasses',
    hasSearch: true,
  },
  runtimeclasses: {
    component: lazy('RuntimeClassesView'),
    dataKey: 'filteredRuntimeClasses',
    hasSearch: true,
  },
  leases: {
    component: lazy('LeasesView'),
    dataKey: 'filteredLeases',
    hasSearch: true,
  },
  serviceaccounts: {
    component: lazy('ServiceAccountsView'),
    dataKey: 'filteredServiceAccounts',
    hasSearch: true,
  },
  roles: {
    component: lazy('RolesView'),
    dataKey: 'filteredRoles',
    hasSearch: true,
  },
  rolebindings: {
    component: lazy('RoleBindingsView'),
    dataKey: 'filteredRoleBindings',
    hasSearch: true,
  },
  clusterroles: {
    component: lazy('ClusterRolesView'),
    dataKey: 'filteredClusterRoles',
    hasSearch: true,
  },
  clusterrolebindings: {
    component: lazy('ClusterRoleBindingsView'),
    dataKey: 'filteredClusterRoleBindings',
    hasSearch: true,
  },
  settings: {
    component: lazy('ClusterSettings'),
  },
}