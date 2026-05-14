<template>
  <div class="rich-text-response-widget">
    <div v-if="htmlContent" class="html-content" v-html="htmlContent"></div>
    <span v-else class="empty-text">-</span>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import type { WidgetComponentProps } from '@/architecture/presentation/widgets/types'
import { sanitizeHtml } from '@/architecture/shared/sanitizeHtml'

const props = withDefaults(defineProps<WidgetComponentProps>(), {
  value: () => ({
    raw: null,
    display: '',
    meta: {},
  }),
})

const htmlContent = computed(() => {
  const fieldValue = props.value ?? (props as any).modelValue
  const raw = fieldValue?.raw

  if (raw === null || raw === undefined || raw === '') {
    return ''
  }

  return sanitizeHtml(String(raw))
})
</script>

<style scoped>
.rich-text-response-widget {
  width: 100%;
}

.html-content {
  width: 100%;
  word-wrap: break-word;
}

.html-content :deep(p) {
  margin: 8px 0;
}

.html-content :deep(h1),
.html-content :deep(h2),
.html-content :deep(h3) {
  margin: 16px 0 8px 0;
  font-weight: bold;
}

.html-content :deep(ul),
.html-content :deep(ol) {
  margin: 8px 0;
  padding-left: 24px;
}

.html-content :deep(blockquote) {
  border-left: 4px solid var(--el-border-color);
  padding-left: 16px;
  margin: 8px 0;
  color: var(--el-text-color-secondary);
}

.html-content :deep(video) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  background-color: #000;
}

.html-content :deep(img) {
  max-width: 100%;
  height: auto;
  border-radius: 4px;
  margin: 8px 0;
  display: block;
  background-color: var(--el-fill-color-lighter);
  object-fit: contain;
}

.html-content :deep(img[src=""]) {
  display: none;
}

.html-content :deep(img:not([src])) {
  display: none;
}

.empty-text {
  color: var(--el-text-color-placeholder);
}
</style>
