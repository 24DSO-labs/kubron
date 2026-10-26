// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import type { InjectionKey, Ref } from 'vue'

/** Имя активного кластера. App.vue делает provide, секции (метрики) делают inject */
export const clusterNameKey: InjectionKey<Ref<string | null | undefined>> = Symbol('clusterName')
