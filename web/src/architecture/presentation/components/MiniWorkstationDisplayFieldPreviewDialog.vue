<template>
  <Teleport to="body">
    <transition name="df-preview-fade">
      <div
        v-if="visible"
        class="df-preview-overlay"
        @click.self="$emit('close')"
        @mousedown.stop
        @mouseup.stop
        @pointerdown.stop
        @pointerup.stop
      >
        <div class="df-preview-modal" @click.stop @mousedown.stop @mouseup.stop @pointerdown.stop @pointerup.stop>
          <div class="df-preview-header">
            <span class="df-preview-title">{{ label }}</span>
            <button class="df-preview-close" @click="$emit('close')" title="关闭">
              <el-icon :size="16"><Close /></el-icon>
            </button>
          </div>
          <div class="df-preview-body">
            <textarea
              :value="content"
              class="df-preview-textarea"
              spellcheck="false"
              @input="$emit('update:content', ($event.target as HTMLTextAreaElement).value)"
            />
          </div>
          <div class="df-preview-footer">
            <span class="df-preview-stats">{{ content.length }} 字符 · {{ lineCount }} 行</span>
            <div class="df-preview-actions">
              <button class="df-preview-btn" @click="$emit('close')">关闭</button>
              <button class="df-preview-btn df-preview-btn--primary" @click="$emit('copy')">
                <el-icon :size="14"><CopyDocument /></el-icon>
                复制全部
              </button>
            </div>
          </div>
        </div>
      </div>
    </transition>
  </Teleport>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Close, CopyDocument } from '@element-plus/icons-vue'

const props = defineProps<{
  visible: boolean
  label: string
  content: string
}>()

defineEmits<{
  (e: 'close'): void
  (e: 'copy'): void
  (e: 'update:content', value: string): void
}>()

const lineCount = computed(() => props.content.split('\n').length)
</script>

<style scoped>
.df-preview-overlay {
  position: fixed;
  inset: 0;
  z-index: 99999;
  background: rgba(0, 0, 0, 0.52);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
}

.df-preview-modal {
  width: 860px;
  max-width: 92vw;
  max-height: 88vh;
  background: var(--el-bg-color, #fff);
  border-radius: 8px;
  box-shadow: 0 16px 48px rgba(0, 0, 0, 0.2);
  display: flex;
  flex-direction: column;
  overflow: hidden;
}

.df-preview-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 16px 20px;
  border-bottom: 1px solid var(--el-border-color-lighter, #eee);
  flex-shrink: 0;
}

.df-preview-title {
  font-size: 16px;
  font-weight: 600;
  color: var(--el-text-color-primary, #303133);
}

.df-preview-close {
  border: none;
  background: none;
  cursor: pointer;
  padding: 4px;
  border-radius: 4px;
  color: var(--el-text-color-secondary, #909399);
  display: flex;
  align-items: center;
  justify-content: center;
  transition: all 0.15s;
}

.df-preview-close:hover {
  background: var(--el-fill-color-light, #f5f7fa);
  color: var(--el-text-color-primary, #303133);
}

.df-preview-body {
  flex: 1;
  min-height: 0;
  padding: 16px 20px;
  overflow: hidden;
}

.df-preview-textarea {
  width: 100%;
  min-height: 360px;
  max-height: calc(88vh - 140px);
  padding: 12px 14px;
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 6px;
  font-family: 'SF Mono', 'Fira Code', 'Consolas', monospace;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-primary, #303133);
  background: var(--el-fill-color-blank, #fff);
  resize: vertical;
  outline: none;
  box-sizing: border-box;
}

.df-preview-textarea:focus {
  border-color: var(--el-color-primary, #409eff);
}

.df-preview-footer {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 20px;
  border-top: 1px solid var(--el-border-color-lighter, #eee);
  flex-shrink: 0;
}

.df-preview-stats {
  font-size: 12px;
  color: var(--el-text-color-secondary, #909399);
}

.df-preview-actions {
  display: flex;
  gap: 8px;
}

.df-preview-btn {
  padding: 8px 16px;
  border: 1px solid var(--el-border-color, #dcdfe6);
  border-radius: 4px;
  background: var(--el-bg-color, #fff);
  color: var(--el-text-color-regular, #606266);
  font-size: 13px;
  cursor: pointer;
  display: inline-flex;
  align-items: center;
  gap: 4px;
  transition: all 0.15s;
}

.df-preview-btn:hover {
  border-color: var(--el-color-primary-light-3, #79bbff);
  color: var(--el-color-primary, #409eff);
}

.df-preview-btn--primary {
  background: var(--el-color-primary, #409eff);
  border-color: var(--el-color-primary, #409eff);
  color: #fff;
}

.df-preview-btn--primary:hover {
  background: var(--el-color-primary-light-3, #79bbff);
  border-color: var(--el-color-primary-light-3, #79bbff);
  color: #fff;
}

.df-preview-fade-enter-active {
  transition: opacity 0.2s ease;
}

.df-preview-fade-leave-active {
  transition: opacity 0.15s ease;
}

.df-preview-fade-enter-from,
.df-preview-fade-leave-to {
  opacity: 0;
}
</style>
