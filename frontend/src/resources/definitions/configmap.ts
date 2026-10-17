import { defineResource } from '../types'
import { callApp } from '../wails'
import { flatNameCol, flatNsCol, ageCol } from './common'
import ConfigMapDataSection from '../../components/resource/sections/ConfigMapDataSection.vue'

export const configMap = defineResource({
  id: 'configmap',
  kind: 'ConfigMap',
  view: 'configmaps',
  title: 'ConfigMaps',
  icon: '📄',
  namespaced: true,
  nav: { group: 'config', label: 'Config Maps' },

  columns: [flatNameCol, flatNsCol, { key: 'dataCount', label: 'Data' }, ageCol],

  properties: [{ label: 'Keys', value: c => c.dataCount }],
  sections: [ConfigMapDataSection],

  api: {
    list: ns => callApp('GetConfigMaps', ns),
    get: (ns, n) => callApp('GetConfigMapDetail', ns, n), // в detail приходит data
    // Нужен Go-метод DeleteConfigMap: до его появления удаление покажет ошибку "недоступен"
    remove: (ns, n) => callApp('DeleteConfigMap', ns, n),
  },
})
