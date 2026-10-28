<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <section class="ov-card certs">
    <h2 class="ov-card-title">{{ $t('overview.certs.title') }}</h2>

    <ul v-if="scan && scan.expiring.length" class="list">
      <li v-for="c in scan.expiring.slice(0, 5)" :key="`${c.namespace}/${c.name}`" class="row" :title="`${c.namespace}/${c.name}`">
        <span class="name">{{ c.subject || c.name }}</span>
        <span class="val" :class="tone(c.daysLeft)">{{ label(c.daysLeft) }}</span>
        <span class="sub">
          {{ c.namespace }}/{{ c.name }} · {{ date(c.notAfter) }}
          <template v-if="c.managed"> · {{ 'cert-manager' }}</template>
        </span>
      </li>
    </ul>
    <p v-else-if="scan" class="none">{{ $t('overview.certs.none', { checked: scan.checked }) }}</p>
    <p v-else class="none">{{ $t('common.loading') }}</p>

    <p v-if="scan && scan.expiring.length > 5" class="more">{{ $t('overview.certs.more', { n: scan.expiring.length - 5 }) }}</p>
  </section>
</template>

<script lang="ts" setup>
import { useI18n } from 'vue-i18n'
import { formatDate } from '../../i18n/format'
import type { CertScan } from '../../types/insights'

defineProps<{ scan: CertScan | null }>()

const { t } = useI18n()

const label = (d: number) =>
  d < 0 ? t('overview.certs.expired', { n: -d }) : d === 0 ? t('overview.certs.today') : t('overview.certs.left', { n: d })
const tone = (d: number) => (d <= 7 ? 'd' : 'w')
const date = (iso: string) => formatDate(iso)
</script>

<style scoped>
.list { list-style: none; margin: 0; padding: 0; display: flex; flex-direction: column; gap: 10px; }
.row { display: grid; grid-template-columns: minmax(0, 1fr) auto; grid-template-areas: 'name val' 'sub sub'; gap: 2px 8px; }
.name { grid-area: name; font-size: 13px; font-weight: 500; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.val { grid-area: val; font-size: 13px; font-weight: 600; }
.sub { grid-area: sub; font-size: 11px; color: var(--ov-muted); overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.w { color: var(--ov-warn); } .d { color: var(--ov-danger); }
.none { margin: 0; font-size: 13px; color: var(--ov-ok); }
.more { margin: 10px 0 0; font-size: 12px; color: var(--ov-muted); }
</style>
