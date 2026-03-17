<template>
  <div class="detail-panel">
    <div class="detail-header">
      <h3>📄 {{cm.name}}</h3>
      <button class="btn-close" @click="$emit('close')">✕</button>
    </div>
    <div class="detail-meta">
      <div class="meta-row"><span class="meta-label">Namespace:</span> {{cm.namespace}}</div>
      <div class="meta-row"><span class="meta-label">Keys:</span> {{cm.dataCount}}</div>
      <div class="meta-row"><span class="meta-label">Age:</span> {{cm.age}}</div>
    </div>
    <div class="detail-section" v-if="cm.data">
      <h4>Data</h4>
      <div v-for="(val,key) in cm.data" :key="key" class="cm-data-item">
        <div class="cm-key">{{key}}</div>
        <pre class="cm-value">{{val}}</pre>
      </div>
    </div>
  </div>
</template>

<script lang="ts" setup>
defineProps<{ cm: any }>()
defineEmits<{ (e: 'close'): void }>()
</script>