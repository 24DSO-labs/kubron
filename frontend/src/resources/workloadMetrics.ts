import { computed, defineComponent, h, inject, ref } from 'vue'
import MetricsBlock from '../components/resource/sections/MetricsBlock.vue'
import PodSetMetricsViewer from '../components/metrics/PodSetMetricsViewer.vue'
import { clusterNameKey } from './context'
import { podOwner, type PodOwnerKind } from './utils'

/** Секция «Metrics» для ресурса, владеющего подами (Deployment, DaemonSet, ...) */
function makeSection(kind: PodOwnerKind) {
  return defineComponent({
    name: `${kind}MetricsSection`,
    props: { resource: { type: Object, required: true } },
    setup(props) {
      const cluster = inject(clusterNameKey, ref(undefined))
      const owner = computed(() => podOwner(kind, String(props.resource?.name ?? '')))
      return () =>
        cluster.value
          ? h(MetricsBlock, null, {
              default: () =>
                h(PodSetMetricsViewer, {
                  clusterName: cluster.value as string,
                  namespace: String(props.resource?.namespace ?? ''),
                  ownerKind: owner.value.kind,
                  ownerRegex: owner.value.regex,
                }),
            })
          : null
    },
  })
}

export const workloadMetrics = {
  deployment: makeSection('deployment'),
  replicaset: makeSection('replicaset'),
  daemonset: makeSection('daemonset'),
  statefulset: makeSection('statefulset'),
  job: makeSection('job'),
}