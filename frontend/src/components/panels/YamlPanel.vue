<template>
  <div class="detail-panel yaml-panel">
    <div class="detail-header">
      <h3>📋 YAML / JSON</h3>
      <div class="yaml-actions">
        <button v-if="!editing" class="btn btn-sm" @click="$emit('startEdit')">✏️ Edit</button>
        <button v-if="editing" class="btn btn-sm" @click="$emit('apply')">💾 Apply</button>
        <button v-if="editing" class="btn btn-sm btn-secondary" @click="$emit('cancelEdit')">Cancel</button>
        <button class="btn-close" @click="$emit('close')">✕</button>
      </div>
    </div>
    <textarea
      v-if="editing"
      :value="content"
      @input="$emit('update:content', ($event.target as HTMLTextAreaElement).value)"
      class="yaml-editor"
    ></textarea>
    <pre v-else class="yaml-output">{{content}}</pre>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ content: string; editing: boolean }>()
defineEmits<{
  (e: 'close'): void
  (e: 'startEdit'): void
  (e: 'apply'): void
  (e: 'cancelEdit'): void
  (e: 'update:content', val: string): void
}>()
</script>