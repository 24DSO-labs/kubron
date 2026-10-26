// Copyright 2026 The Kubron Authors
// SPDX-License-Identifier: Apache-2.0

import { defineResource } from '../types'
import type { k8s } from '../../../wailsjs/go/models'
import { callApp } from '../wails'
import { flatNameCol, ageCol, controlledByProp, managedByProp } from './common'
import WebhooksSection from '../../components/resource/sections/WebhooksSection.vue'

/** Mutating и Validating конфигурации отличаются только kind и методом списка */
const webhookConfig = (opts: {
  id: string
  kind: string
  view: string
  title: string
  label: string
  list: 'GetMutatingWebhookConfigurations' | 'GetValidatingWebhookConfigurations'
}) =>
  defineResource<k8s.WebhookConfigurationInfo>({
    id: opts.id,
    kind: opts.kind,
    view: opts.view,
    title: opts.title,
    namespaced: false,
    nav: { group: 'config', label: opts.label },

    autoRefresh: true,

    columns: [
      flatNameCol,
      { key: 'webhookCount', label: 'Webhooks' },
      ageCol,
    ],

    properties: [
      { label: 'Webhooks', value: (c: any) => c.webhookCount },
      { label: 'Finalizers', value: (c: any) => (c.finalizers ?? []).join(', ') },
      controlledByProp,
      managedByProp,
    ],
    sections: [WebhooksSection],

    api: {
      list: () => callApp(opts.list),
      remove: (_ns, name) => callApp('DeleteResource', opts.id, '', name),
    },
  })

export const mutatingWebhookConfiguration = webhookConfig({
  id: 'mutatingwebhookconfiguration',
  kind: 'MutatingWebhookConfiguration',
  view: 'mutating-webhooks',
  title: 'MutatingWebhookConfigurations',
  label: 'Mutating Webhook Configs',
  list: 'GetMutatingWebhookConfigurations',
})

export const validatingWebhookConfiguration = webhookConfig({
  id: 'validatingwebhookconfiguration',
  kind: 'ValidatingWebhookConfiguration',
  view: 'validating-webhooks',
  title: 'ValidatingWebhookConfigurations',
  label: 'Validating Webhook Configs',
  list: 'GetValidatingWebhookConfigurations',
})
