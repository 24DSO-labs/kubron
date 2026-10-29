<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <label class="lang-switcher">
    <span class="lang-switcher-label">{{ $t('common.language') }}</span>
    <select class="lang-switcher-select" :value="current" @change="onChange">
      <option v-for="lang in LANGUAGES" :key="lang.code" :value="lang.code">{{ lang.name }}</option>
    </select>
  </label>
</template>

<script lang="ts" setup>
import { LANGUAGES } from '../i18n/languages'
import { locale, setLocale } from '../i18n'

// Названия языков не переводятся: человек, выбравший не тот язык, должен найти свой.
const current = locale

function onChange(event: Event) {
  void setLocale((event.target as HTMLSelectElement).value)
}
</script>

<style scoped>
.lang-switcher { display: inline-flex; align-items: center; gap: 8px; font-size: 13px; color: #a6adc8; }
.lang-switcher-select {
  background: #11111b;
  border: 1px solid #313244;
  border-radius: 8px;
  padding: 6px 10px;
  font-size: 13px;
  color: #cdd6f4;
}
.lang-switcher-select:focus { outline: none; border-color: #89b4fa; box-shadow: 0 0 0 3px rgba(137, 180, 250, 0.18); }
</style>
