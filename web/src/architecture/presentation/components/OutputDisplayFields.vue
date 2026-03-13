<!--
  OutputDisplayFields - 工具调用结果中的「提升展示」字段
  从 run_form_submit 等工具的 output_display 参数提取，展示在文件列表旁，方便用户查看和复制。
  支持预览弹窗（可编辑 + 一键复制）。
-->
<template>
  <div v-if="fields.length > 0" class="output-display-fields">
    <div class="odf-head">
      <el-icon><Memo /></el-icon>
      <span>输出数据</span>
    </div>
    <div class="odf-list">
      <div v-for="(field, idx) in fields" :key="idx" class="odf-card">
        <div class="odf-card-header">
          <span class="odf-label">{{ field.label }}</span>
          <div class="odf-header-actions">
            <el-button link size="small" class="odf-action-btn" @click="openPreview(field)">
              <el-icon :size="12"><View /></el-icon>
              预览
            </el-button>
            <el-button link size="small" class="odf-action-btn" @click="copyValue(field)">
              <el-icon :size="12"><CopyDocument /></el-icon>
              复制
            </el-button>
          </div>
        </div>
        <div :class="['odf-value', { 'odf-value--collapsed': !expandedSet.has(idx) && isLong(field) }]">
          <pre class="odf-pre">{{ field.value }}</pre>
        </div>
        <el-button
          v-if="isLong(field)"
          link
          size="small"
          class="odf-expand-btn"
          @click="toggleExpand(idx)"
        >
          {{ expandedSet.has(idx) ? '收起' : '展开全部' }}
          <el-icon :size="12"><component :is="expandedSet.has(idx) ? ArrowUp : ArrowDown" /></el-icon>
        </el-button>
      </div>
    </div>

    <!-- 预览弹窗：Teleport 到 body，z-index 99999 -->
    <Teleport to="body">
      <transition name="odf-preview-fade">
        <div
          v-if="previewVisible"
          class="df-preview-overlay"
          @click.self="closePreview"
          @mousedown.stop
          @mouseup.stop
          @pointerdown.stop
          @pointerup.stop
        >
          <div class="df-preview-modal" @click.stop @mousedown.stop @mouseup.stop @pointerdown.stop @pointerup.stop>
            <div class="df-preview-header">
              <span class="df-preview-title">{{ previewLabel }}</span>
              <button class="df-preview-close" @click="closePreview" title="关闭">
                <el-icon :size="16"><Close /></el-icon>
              </button>
            </div>
            <div class="df-preview-body">
              <textarea
                v-model="previewContent"
                class="df-preview-textarea"
                spellcheck="false"
              />
            </div>
            <div class="df-preview-footer">
              <span class="df-preview-stats">{{ previewContent.length }} 字符 · {{ previewContent.split('\n').length }} 行</span>
              <div class="df-preview-actions">
                <button class="df-preview-btn" @click="closePreview">关闭</button>
                <button class="df-preview-btn df-preview-btn--primary" @click="copyPreviewContent">
                  <el-icon :size="14"><CopyDocument /></el-icon>
                  复制全部
                </button>
              </div>
            </div>
          </div>
        </div>
      </transition>
    </Teleport>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { Memo, CopyDocument, ArrowDown, ArrowUp, View, Close } from '@element-plus/icons-vue'
import { ElMessage } from 'element-plus'
import type { OutputDisplayField } from '@/architecture/presentation/composables/useOutputDisplayFields'

defineProps<{
  fields: OutputDisplayField[]
}>()

const LONG_THRESHOLD = 200

const expandedSet = ref<Set<number>>(new Set())

const previewVisible = ref(false)
const previewLabel = ref('')
const previewContent = ref('')

function isLong(field: OutputDisplayField): boolean {
  return field.value.length > LONG_THRESHOLD || field.value.split('\n').length > 5
}

function toggleExpand(idx: number) {
  const s = new Set(expandedSet.value)
  if (s.has(idx)) s.delete(idx)
  else s.add(idx)
  expandedSet.value = s
}

function openPreview(field: OutputDisplayField) {
  previewLabel.value = field.label
  previewContent.value = field.value
  previewVisible.value = true
}

function closePreview() {
  setTimeout(() => { previewVisible.value = false }, 0)
}

async function copyValue(field: OutputDisplayField) {
  try {
    await navigator.clipboard.writeText(field.value)
    ElMessage.success(`已复制「${field.label}」`)
  } catch {
    ElMessage.error('复制失败')
  }
}

async function copyPreviewContent() {
  try {
    await navigator.clipboard.writeText(previewContent.value)
    ElMessage.success(`已复制「${previewLabel.value}」`)
  } catch {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped lang="scss">
.output-display-fields {
  margin-top: 10px;
}

.odf-head {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;

  .el-icon {
    font-size: 14px;
    color: var(--el-color-primary);
  }
}

.odf-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.odf-card {
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-small);
  overflow: hidden;
}

.odf-card-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 8px 12px;
  background: var(--el-fill-color-light);
  border-bottom: 1px solid var(--el-border-color-extra-light);
}

.odf-label {
  font-size: 12px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.odf-header-actions {
  display: flex;
  gap: 8px;
  align-items: center;
}

.odf-action-btn {
  font-size: 12px;
  gap: 3px;
}

.odf-value {
  padding: 8px 12px;
  position: relative;

  &.odf-value--collapsed {
    max-height: 7.5em;
    overflow: hidden;

    &::after {
      content: '';
      position: absolute;
      bottom: 0;
      left: 0;
      right: 0;
      height: 2.5em;
      background: linear-gradient(transparent, var(--el-fill-color-lighter));
      pointer-events: none;
    }
  }
}

.odf-pre {
  margin: 0;
  white-space: pre-wrap;
  word-break: break-word;
  font-size: 12px;
  line-height: 1.6;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  color: var(--el-text-color-regular);
}

.odf-expand-btn {
  display: flex;
  align-items: center;
  gap: 3px;
  width: 100%;
  justify-content: center;
  padding: 4px 0;
  font-size: 12px;
  border-top: 1px solid var(--el-border-color-extra-light);
  color: var(--el-color-primary);
}

.odf-preview-fade-enter-active {
  transition: opacity 0.2s ease;
}
.odf-preview-fade-leave-active {
  transition: opacity 0.15s ease;
}
.odf-preview-fade-enter-from,
.odf-preview-fade-leave-to {
  opacity: 0;
}
</style>
