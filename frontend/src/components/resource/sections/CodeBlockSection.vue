<!-- Copyright 2026 The Kubron Authors -->
<!-- SPDX-License-Identifier: Apache-2.0 -->
<template>
  <div class="rd-row" :class="{ 'rd-click': lines.length }" @click="toggle">
    <span class="rd-label">{{ title }}<span v-if="lines.length" class="rd-arrow" :class="{ open }">›</span></span>
    <span v-if="!lines.length" class="rd-muted">—</span>
    <span v-else-if="!open" class="rd-muted">{{ groups }} {{ groups === 1 ? 'Rule' : 'Rules' }}</span>
  </div>

  <div v-if="open && lines.length" class="rd-code">
    <div class="rd-code-gutter">
      <div v-for="line in visible" :key="'g' + line.idx" class="rd-code-gline">
        <span class="rd-code-num">{{ line.num }}</span>
        <span
          v-if="line.foldable"
          class="rd-code-fold"
          :class="{ collapsed: line.collapsed }"
          @click.stop="toggleFold(line.idx)"
        >›</span>
        <span v-else class="rd-code-spacer"></span>
      </div>
    </div>
    <div class="rd-code-border"></div>
    <div class="rd-code-body">
      <div v-for="line in visible" :key="'c' + line.idx" class="rd-code-line">
        <span v-if="line.indent > 0" class="rd-code-indent">
          <span v-for="level in line.indent" :key="level" class="rd-code-slot">
            <span v-if="line.guides.includes(level - 1)" class="rd-code-guide"></span>
            <span class="sp">&nbsp;&nbsp;</span>
          </span>
        </span>
        <span class="rd-code-text" v-html="line.html"></span>
      </div>
    </div>
  </div>
</template>

<script lang="ts">
// Два корневых узла: не пробрасываем лишние атрибуты (@open от панели) на фрагмент
export default { inheritAttrs: false }
</script>

<script lang="ts" setup>
/** Блок кода (YAML-подобный) со сворачиванием: используется для Affinity */
import { ref, computed } from 'vue'

const props = defineProps<{ title: string; lines: string[] }>()

const open = ref(false)
const collapsed = ref<Set<number>>(new Set())

function toggle() {
  if (props.lines.length) open.value = !open.value
}
function toggleFold(idx: number) {
  const s = new Set(collapsed.value)
  if (s.has(idx)) s.delete(idx)
  else s.add(idx)
  collapsed.value = s
}

/** Число правил верхнего уровня */
const groups = computed(() => props.lines.filter(l => l.trim() && !l.startsWith(' ')).length)

interface Node { raw: string; indent: number; children: Node[] }
interface Line { html: string; num: number; foldable: boolean; collapsed: boolean; idx: number; indent: number; guides: number[] }

const indentOf = (line: string) => Math.floor((line.match(/^(\s*)/)?.[1].length ?? 0) / 2)
const esc = (s: unknown) => String(s).replace(/&/g, '&amp;').replace(/</g, '&lt;').replace(/>/g, '&gt;')

function highlight(raw: string): string {
  if (!raw.trim()) return '&nbsp;'
  let rest = raw.trimStart()
  let out = ''
  if (rest.startsWith('- ')) {
    out += '<span class="hl-dash">-&nbsp;</span>'
    rest = rest.slice(2)
  }
  const ci = rest.indexOf(': ')
  if (ci > 0) {
    out += `<span class="hl-key">${esc(rest.slice(0, ci))}</span><span class="hl-colon">:&nbsp;</span><span class="hl-val">${esc(rest.slice(ci + 2))}</span>`
  } else if (rest.endsWith(':')) {
    out += `<span class="hl-key">${esc(rest.slice(0, -1))}</span><span class="hl-colon">:</span>`
  } else {
    out += `<span class="hl-val">${esc(rest)}</span>`
  }
  return out
}

function buildTree(lines: string[]): Node[] {
  const nodes: Node[] = lines.map(raw => ({ raw, indent: indentOf(raw), children: [] }))
  const roots: Node[] = []
  const stack: Node[] = []
  for (const node of nodes) {
    while (stack.length && stack[stack.length - 1].indent >= node.indent) stack.pop()
    if (!stack.length) roots.push(node)
    else stack[stack.length - 1].children.push(node)
    if (node.raw.trimEnd().endsWith(':')) stack.push(node)
  }
  return roots
}

function flatten(nodes: Node[], counter: { i: number }, folded: Set<number>): Line[] {
  const out: Line[] = []
  for (const node of nodes) {
    const idx = counter.i++
    const foldable = node.children.length > 0
    const isFolded = folded.has(idx)
    out.push({ html: highlight(node.raw), num: idx + 1, foldable, collapsed: isFolded, idx, indent: node.indent, guides: [] })
    if (!isFolded && foldable) out.push(...flatten(node.children, counter, folded))
  }
  return out
}

function computeGuides(lines: Line[]) {
  for (let i = 0; i < lines.length; i++) {
    const guides: number[] = []
    for (let level = 0; level < lines[i].indent; level++) {
      for (let j = i + 1; j < lines.length; j++) {
        if (lines[j].indent <= level) break
        guides.push(level)
        break
      }
    }
    lines[i].guides = guides
  }
}

const visible = computed<Line[]>(() => {
  const flat = flatten(buildTree(props.lines), { i: 0 }, collapsed.value)
  computeGuides(flat)
  return flat
})
</script>
