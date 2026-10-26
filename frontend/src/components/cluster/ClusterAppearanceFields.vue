<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div :style="{ '--cl-pick': color || DEFAULT_CLUSTER_COLOR }">
    <div class="cl-field">
      <span :id="`${uid}-env`" class="cl-label">Окружение</span>
      <div class="cl-segment" role="radiogroup" :aria-labelledby="`${uid}-env`">
        <button v-for="e in CLUSTER_ENVIRONMENTS" :key="e.key || 'none'" type="button" role="radio"
          class="cl-segment-item" :class="{ on: environment === e.key }" :aria-checked="environment === e.key"
          @click="emit('update:environment', e.key)">
          <i v-if="e.key" class="cl-dot" :style="{ background: e.color }"></i>{{ e.label }}
        </button>
      </div>
      <p v-if="environment === 'prod'" class="cl-hint">
        Рабочий кластер: пока он открыт, в шапке приложения горит красная линия — его не спутать с другим.
      </p>
    </div>

    <div class="cl-field">
      <span :id="`${uid}-glyph`" class="cl-label">Значок</span>
      <div class="cl-glyphs" role="radiogroup" :aria-labelledby="`${uid}-glyph`">
        <button type="button" role="radio" class="cl-glyph" :class="{ on: !glyphKey }" :aria-checked="!glyphKey"
          title="Монограмма из названия" @click="emit('update:icon', '')">
          <span class="cl-glyph-mono">{{ initials }}</span>
        </button>
        <button v-for="g in CLUSTER_GLYPHS" :key="g.key" type="button" role="radio" class="cl-glyph"
          :class="{ on: glyphKey === g.key }" :aria-checked="glyphKey === g.key" :title="g.label"
          @click="emit('update:icon', g.key)">
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="1.8" stroke-linecap="round"
            stroke-linejoin="round">
            <path v-for="d in g.paths" :key="d" :d="d" />
          </svg>
        </button>
      </div>
    </div>

    <div class="cl-field">
      <span :id="`${uid}-color`" class="cl-label">Цвет</span>
      <div class="cl-swatches" role="radiogroup" :aria-labelledby="`${uid}-color`">
        <button v-for="c in swatches" :key="c" type="button" role="radio" class="cl-swatch"
          :class="{ on: sameColor(c) }" :style="{ background: c }" :aria-checked="sameColor(c)" :aria-label="c"
          @click="emit('update:color', c)"></button>
        <label class="cl-swatch cl-swatch-custom" title="Свой цвет">
          <input type="color" :value="color || DEFAULT_CLUSTER_COLOR" aria-label="Свой цвет" @input="onCustomColor" />
        </label>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
import { computed, useId } from 'vue'
import './cluster.css'
import {
  CLUSTER_COLORS, CLUSTER_ENVIRONMENTS, CLUSTER_GLYPHS, DEFAULT_CLUSTER_COLOR,
  clusterInitials, findGlyph,
} from './clusterStyle'

// Окружение, значок и цвет кластера. Название редактирует диалог: здесь оно нужно только для монограммы.
const props = defineProps<{
  name: string
  icon: string
  color: string
  environment: string
}>()

const emit = defineEmits<{
  (e: 'update:icon', v: string): void
  (e: 'update:color', v: string): void
  (e: 'update:environment', v: string): void
}>()

const uid = useId()

const initials = computed(() => clusterInitials(props.name))
// Устаревший эмодзи считается «монограммой»: выбранным подсвечивается именно она
const glyphKey = computed(() => findGlyph(props.icon)?.key ?? '')

const sameColor = (c: string) => c.toLowerCase() === (props.color || DEFAULT_CLUSTER_COLOR).toLowerCase()

// Цвет, которого нет в палитре (старый или выбранный вручную), показывается первым, чтобы его можно было вернуть
const swatches = computed(() => {
  const current = (props.color || '').toLowerCase()
  return current && !CLUSTER_COLORS.includes(current) ? [current, ...CLUSTER_COLORS] : CLUSTER_COLORS
})

function onCustomColor(e: Event) {
  emit('update:color', (e.target as HTMLInputElement).value.toLowerCase())
}
</script>
