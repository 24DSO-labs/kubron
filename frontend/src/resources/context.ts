import type { InjectionKey, Ref } from 'vue'

/** Имя активного кластера. App.vue делает provide, секции (метрики) делают inject */
export const clusterNameKey: InjectionKey<Ref<string | null | undefined>> = Symbol('clusterName')
