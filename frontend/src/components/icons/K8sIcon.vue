<template>
  <img
    v-if="iconSrc"
    :src="iconSrc"
    :alt="alt || name"
    :class="['k8s-icon', sizeClass, { 'icon-danger': danger }]"
    :style="iconStyle"
  />
</template>

<script lang="ts" setup>
import { computed } from 'vue'

import createIcon from '../../assets/icons/create.svg'
import cloneIcon from '../../assets/icons/clone.svg'
import editIcon from '../../assets/icons/edit.svg'
import deleteIcon from '../../assets/icons/delete.svg'
import viewIcon from '../../assets/icons/view.svg'
import viewOffIcon from '../../assets/icons/view-off.svg'
import refreshIcon from '../../assets/icons/refresh.svg'
import exportIcon from '../../assets/icons/export.svg'
import playIcon from '../../assets/icons/play.svg'
import stopIcon from '../../assets/icons/stop.svg'
import restartIcon from '../../assets/icons/restart.svg'
import pauseIcon from '../../assets/icons/pause.svg'
import rollbackIcon from '../../assets/icons/rollback.svg'
import scaleIcon from '../../assets/icons/scale.svg'
import scaleUpIcon from '../../assets/icons/scale-up.svg'
import scaleDownIcon from '../../assets/icons/scale-down.svg'
import scaleZeroIcon from '../../assets/icons/scale-zero.svg'
import autoscaleIcon from '../../assets/icons/autoscale.svg'
import openBrowserIcon from '../../assets/icons/open-browser.svg'
import portForwardIcon from '../../assets/icons/port-forward.svg'
import ingressIcon from '../../assets/icons/ingress.svg'
import serviceIcon from '../../assets/icons/service.svg'
import endpointIcon from '../../assets/icons/endpoint.svg'
import yamlIcon from '../../assets/icons/yaml.svg'
import applyIcon from '../../assets/icons/apply.svg'
import importIcon from '../../assets/icons/import.svg'
import configmapIcon from '../../assets/icons/configmap.svg'
import secretIcon from '../../assets/icons/secret.svg'
import volumeMountIcon from '../../assets/icons/volume-mount.svg'
import logsIcon from '../../assets/icons/logs.svg'
import followLogsIcon from '../../assets/icons/follow-logs.svg'
import shellIcon from '../../assets/icons/shell.svg'
import execIcon from '../../assets/icons/exec.svg'
import diagnosticsIcon from '../../assets/icons/diagnostics.svg'
import eventsIcon from '../../assets/icons/events.svg'
import serviceAccountIcon from '../../assets/icons/service-account.svg'
import assignRoleIcon from '../../assets/icons/assign-role.svg'
import clusterRoleIcon from '../../assets/icons/cluster-role.svg'
import bindRoleIcon from '../../assets/icons/bind-role.svg'
import viewAccessIcon from '../../assets/icons/view-access.svg'
import networkPolicyIcon from '../../assets/icons/network-policy.svg'
import pvcIcon from '../../assets/icons/pvc.svg'
import pvcBindIcon from '../../assets/icons/pvc-bind.svg'
import pvcExpandIcon from '../../assets/icons/pvc-expand.svg'
import unmountIcon from '../../assets/icons/unmount.svg'
import storageClassIcon from '../../assets/icons/storage-class.svg'
import copyNameIcon from '../../assets/icons/copy-name.svg'
import copyYamlIcon from '../../assets/icons/copy-yaml.svg'
import addLabelIcon from '../../assets/icons/add-label.svg'
import removeLabelIcon from '../../assets/icons/remove-label.svg'
import dependenciesIcon from '../../assets/icons/dependencies.svg'
import documentationIcon from '../../assets/icons/documentation.svg'
import namespaceIcon from '../../assets/icons/namespace.svg'
import overviewIcon from '../../assets/icons/overview.svg'
import settingsIcon from '../../assets/icons/settings.svg'
import attachIcon from '../../assets/icons/attach.svg'
import closeIcon from '../../assets/icons/close.svg'
import copyIcon from '../../assets/icons/copy.svg'
import downloadIcon from '../../assets/icons/download.svg'
import evictIcon from '../../assets/icons/evict.svg'
import terminalIcon from '../../assets/icons/terminal.svg'
import timestampIcon from '../../assets/icons/timestamp.svg'
import wrapLinesIcon from '../../assets/icons/wrap-lines.svg'
import starIcon from '../../assets/icons/star.svg'

const ICON_MAP: Record<string, string> = {
  create: createIcon, clone: cloneIcon, edit: editIcon,
  delete: deleteIcon, view: viewIcon, 'view-off': viewOffIcon, refresh: refreshIcon,
  export: exportIcon, play: playIcon, start: playIcon,
  stop: stopIcon, restart: restartIcon, pause: pauseIcon,
  resume: playIcon, recreate: restartIcon, rollback: rollbackIcon,
  scale: scaleIcon, 'scale-up': scaleUpIcon, 'scale-down': scaleDownIcon,
  'scale-zero': scaleZeroIcon, autoscale: autoscaleIcon,
  'open-browser': openBrowserIcon, 'port-forward': portForwardIcon,
  ingress: ingressIcon, service: serviceIcon, endpoint: endpointIcon,
  endpoints: endpointIcon, yaml: yamlIcon, 'edit-yaml': editIcon,
  'view-yaml': yamlIcon, apply: applyIcon, import: importIcon,
  configmap: configmapIcon, secret: secretIcon, 'volume-mount': volumeMountIcon,
  logs: logsIcon, 'follow-logs': followLogsIcon, shell: shellIcon,
  exec: execIcon, 'port-forwarding': portForwardIcon,
  diagnostics: diagnosticsIcon, events: eventsIcon,
  'service-account': serviceAccountIcon, 'assign-role': assignRoleIcon,
  'cluster-role': clusterRoleIcon, 'bind-role': bindRoleIcon,
  'view-access': viewAccessIcon, 'network-policy': networkPolicyIcon,
  pvc: pvcIcon, 'pvc-bind': pvcBindIcon, 'pvc-expand': pvcExpandIcon,
  unmount: unmountIcon, 'storage-class': storageClassIcon,
  'copy-name': copyNameIcon, 'copy-yaml': copyYamlIcon,
  'add-label': addLabelIcon, 'remove-label': removeLabelIcon,
  dependencies: dependenciesIcon, documentation: documentationIcon,
  namespace: namespaceIcon, overview: overviewIcon, settings: settingsIcon,
  attach: attachIcon, close: closeIcon, copy: copyIcon,
  download: downloadIcon, evict: evictIcon, terminal: terminalIcon,
  timestamp: timestampIcon, 'wrap-lines': wrapLinesIcon,
  star: starIcon,
}

const props = defineProps<{
  name: string
  alt?: string
  size?: 'sm' | 'md' | 'lg'
  danger?: boolean
  color?: string
}>()

const iconSrc = computed(() => ICON_MAP[props.name] || '')
const sizeClass = computed(() => {
  switch (props.size) {
    case 'sm': return 'icon-sm'
    case 'lg': return 'icon-lg'
    default: return 'icon-md'
  }
})
const iconStyle = computed(() => {
  const s: Record<string, string> = {}
  if (props.color) s.color = props.color
  return s
})
</script>

<style scoped>
.k8s-icon {
  display: inline-block;
  vertical-align: middle;
  flex-shrink: 0;
  transition: opacity 0.15s;
  /* Все иконки белые по умолчанию */
  filter: invert(1);
  opacity: 0.9;
}
.icon-sm { width: 16px; height: 16px; }
.icon-md { width: 20px; height: 20px; }
.icon-lg { width: 24px; height: 24px; }
.icon-danger {
  filter: invert(31%) sepia(91%) saturate(6000%) hue-rotate(350deg) brightness(95%) contrast(90%);
  opacity: 1;
}
</style>
