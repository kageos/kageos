<template>
  <div
    v-if="hasContent"
    class="select-fuzzy-presentation"
    :class="{ 'is-compact': compact, 'has-files-only': files && !richText }"
  >
    <div v-if="richText" class="presentation-rich-text-container" :class="{ 'is-expanded': isExpanded, 'is-collapsible': shouldShowToggle }">
      <div class="presentation-rich-text" ref="richTextRef">
        <RichTextResponseWidget
          :field="richTextField"
          :value="richTextValue"
          mode="detail"
          field-path="__select_fuzzy_rich_text"
        />
      </div>
      <div v-if="shouldShowToggle && !isExpanded" class="rich-text-fade-mask"></div>
      <div v-if="shouldShowToggle" class="rich-text-toggle-btn" @click.stop="toggleExpand">
        <span class="toggle-text">{{ isExpanded ? '收起' : '展开全文' }}</span>
        <el-icon class="toggle-icon"><ArrowDown v-if="!isExpanded" /><ArrowUp v-else /></el-icon>
      </div>
    </div>
    <div v-if="files" class="presentation-files">
      <FilesWidget
        :field="filesField"
        :value="filesValue"
        mode="table-cell"
        field-path="__select_fuzzy_files"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, onMounted, nextTick, watch } from 'vue'
import { ArrowDown, ArrowUp } from '@element-plus/icons-vue'
import type { FieldConfig, FieldValue } from '@/architecture/presentation/widgets/types'
import FilesWidget from './FilesWidget.vue'
import RichTextResponseWidget from './RichTextResponseWidget.vue'

const props = withDefaults(defineProps<{
  richText?: string
  files?: string
  compact?: boolean
}>(), {
  richText: '',
  files: '',
  compact: false,
})

const hasContent = computed(() => Boolean(props.richText || props.files))

const isExpanded = ref(false)
const shouldShowToggle = ref(false)
const richTextRef = ref<HTMLElement | null>(null)

const MAX_COMPACT_HEIGHT = 80 // 候选项中的最大高度
const MAX_DETAIL_HEIGHT = 160 // 选中详情中的最大高度

const checkHeight = () => {
  if (!richTextRef.value || !props.richText) {
    shouldShowToggle.value = false
    return
  }
  
  const maxHeight = props.compact ? MAX_COMPACT_HEIGHT : MAX_DETAIL_HEIGHT
  // 预留一些缓冲，避免刚刚好超出一丁点也显示展开
  shouldShowToggle.value = richTextRef.value.scrollHeight > maxHeight + 10
}

onMounted(() => {
  nextTick(checkHeight)
})

watch(() => props.richText, () => {
  nextTick(checkHeight)
})

function toggleExpand() {
  isExpanded.value = !isExpanded.value
}

const richTextField = {
  code: '__select_fuzzy_rich_text',
  name: '详细说明',
  widget: { type: 'richtext', config: {} },
} as FieldConfig

const filesField = {
  code: '__select_fuzzy_files',
  name: '相关文件',
  widget: {
    type: 'files',
    config: { thumbnail: true, list_preview: true },
  },
} as FieldConfig

const richTextValue = computed<FieldValue>(() => ({
  raw: props.richText,
  display: props.richText,
  meta: {},
}))

const filesValue = computed<FieldValue>(() => ({
  raw: props.files,
  display: props.files ? '相关文件' : '',
  meta: {},
}))
</script>

<style scoped>
.select-fuzzy-presentation {
  display: grid;
  gap: 8px;
  margin-top: 8px;
  padding: 10px;
  border-radius: 6px;
  background: color-mix(in srgb, var(--el-color-primary) 3%, transparent);
  border: 1px solid color-mix(in srgb, var(--el-color-primary) 8%, var(--el-border-color-lighter));
  color: var(--el-text-color-regular);
  font-size: 13px;
}

.select-fuzzy-presentation.is-compact {
  margin-top: 6px;
  padding: 8px 10px;
  border: none;
  background: transparent;
  border-left: 2px solid color-mix(in srgb, var(--el-color-primary) 30%, var(--el-border-color-light));
  border-radius: 0 4px 4px 0;
}

.select-fuzzy-presentation.is-compact.has-files-only {
  margin: 0;
  padding: 0;
  border: none;
  background: transparent;
}


.presentation-rich-text-container {
  position: relative;
  overflow: hidden;
}

.presentation-rich-text-container.is-collapsible:not(.is-expanded) .presentation-rich-text {
  max-height: 160px; /* 选中详情的默认高度 */
  overflow: hidden;
}

.is-compact .presentation-rich-text-container.is-collapsible:not(.is-expanded) .presentation-rich-text {
  max-height: 80px; /* 候选项的默认高度，约 3-4 行 */
}

.rich-text-fade-mask {
  position: absolute;
  bottom: 24px;
  left: 0;
  right: 0;
  height: 40px;
  background: linear-gradient(to bottom, transparent, var(--el-bg-color));
  pointer-events: none;
}

.select-fuzzy-presentation.is-compact .rich-text-fade-mask {
  /* 候选项背景通常带有悬浮色，使用一种近似的渐变或者非常轻的主题色渐变 */
  background: linear-gradient(to bottom, transparent, color-mix(in srgb, var(--el-bg-color) 80%, var(--el-fill-color-light)));
}

/* 注意：这里的全局选择器仅为了在弹出层中复写渐变颜色 */
:global(.suggestion-item:hover .select-fuzzy-presentation.is-compact .rich-text-fade-mask),
:global(.suggestion-item.active .select-fuzzy-presentation.is-compact .rich-text-fade-mask) {
  background: linear-gradient(to bottom, transparent, var(--el-fill-color-light));
}

:global(.suggestion-item.selected .select-fuzzy-presentation.is-compact .rich-text-fade-mask) {
  background: linear-gradient(to bottom, transparent, color-mix(in srgb, var(--el-color-primary) 12%, var(--el-bg-color)));
}

.rich-text-toggle-btn {
  display: flex;
  align-items: center;
  justify-content: center;
  gap: 4px;
  height: 24px;
  color: var(--el-color-primary);
  font-size: 12px;
  cursor: pointer;
  margin-top: 4px;
  transition: opacity 0.2s;
}

.rich-text-toggle-btn:hover {
  opacity: 0.8;
}

.presentation-rich-text :deep(.rich-text-response-widget) {
  font-size: 13px;
  line-height: 1.5;
}

.presentation-rich-text :deep(.html-content p),
.presentation-rich-text :deep(.html-content ul),
.presentation-rich-text :deep(.html-content ol) {
  margin: 4px 0;
}

.presentation-rich-text :deep(.html-content h1),
.presentation-rich-text :deep(.html-content h2),
.presentation-rich-text :deep(.html-content h3) {
  margin: 8px 0 4px 0;
  font-size: 14px;
}

.presentation-rich-text :deep(.html-content img) {
  max-height: 120px; /* 限制图片高度，避免过高 */
  width: auto;
}

.presentation-files {
  min-width: 0;
  margin-top: 4px;
}

.is-compact .presentation-files {
  margin-top: 0;
}
</style>