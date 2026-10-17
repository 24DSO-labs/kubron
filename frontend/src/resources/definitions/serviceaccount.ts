import { defineResource } from '../types'
import { callApp } from '../wails'
import { nameCol, nsCol, ageCol } from './common'

export const serviceAccount = defineResource({
  id: 'serviceaccount',
  kind: 'ServiceAccount',
  view: 'serviceaccounts',
  title: 'Service Accounts',
  icon: 'service-account',
  namespaced: true,
  nav: { group: 'access', label: 'Service Accounts' },
  columns: [nameCol, nsCol, ageCol],
  // TODO: сверить поля с ServiceAccountDetailPanel (secrets / imagePullSecrets)
  properties: [
    { label: 'Secrets', value: r => r.secrets?.length ?? 0 },
    { label: 'Image pull secrets', value: r => r.imagePullSecrets?.length ?? 0 },
  ],
  api: {
    list: ns => callApp('GetServiceAccounts', ns),
    get: (ns, n) => callApp('GetServiceAccountDetail', ns, n),
    remove: (ns, n) => callApp('DeleteServiceAccount', ns, n),
  },
})
