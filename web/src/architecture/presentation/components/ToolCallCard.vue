<!--
  ToolCallCard - 工具调用记录卡片组件
  类似 Cursor 的工具调用显示，支持展开/折叠，显示详细信息
-->
<template>
  <div :class="['tool-call-card', { 'tool-call-card--expanded': expanded, 'tool-call-card--error': toolCall.status === 'error', 'tool-call-card--running': toolCall.status === 'running', 'tool-call-card--streaming': toolCall.status === 'streaming' }]">
    <div class="tool-call-header" @click="toggleExpand">
      <div class="tool-call-info">
        <el-icon :class="['tool-call-icon', toolCall.status === 'ok' ? 'success' : toolCall.status === 'running' ? 'running' : toolCall.status === 'streaming' ? 'streaming' : 'error']">
          <Check v-if="toolCall.status === 'ok'" />
          <Loading v-else-if="toolCall.status === 'running'" class="is-loading" />
          <Loading v-else-if="toolCall.status === 'streaming'" class="is-loading" />
          <Close v-else />
        </el-icon>
        <span class="tool-call-name">{{ toolCall.name }}</span>
        <el-tag v-if="toolCall.status === 'streaming'" type="info" size="small" class="tool-call-status">解析中</el-tag>
        <el-tag v-else-if="toolCall.status === 'running'" type="info" size="small" class="tool-call-status">执行中</el-tag>
        <el-tag v-else :type="toolCall.status === 'ok' ? 'success' : 'danger'" size="small" class="tool-call-status">
          {{ toolCall.status === 'ok' ? '成功' : '失败' }}
        </el-tag>
      </div>
      <el-icon class="expand-icon" :class="{ 'expanded': expanded }">
        <ArrowDown />
      </el-icon>
    </div>
    <div v-if="expanded" class="tool-call-details">
      <!-- 参数显示：有值则展示并支持复制，无值则显示占位 -->
      <div class="tool-call-section">
        <div class="section-title">
          <el-icon><Document /></el-icon>
          <span>参数</span>
          <el-button v-if="toolCall.arguments" text size="small" @click.stop="copyArguments" class="copy-btn">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
        </div>
        <div class="section-content">
          <pre v-if="toolCall.arguments" class="json-content">{{ formatJSON(toolCall.arguments) }}</pre>
          <pre v-else class="json-content json-content--empty">（无参数或加载中）</pre>
        </div>
      </div>
      <!-- 结果显示 -->
      <div v-if="toolCall.result" class="tool-call-section">
        <div class="section-title">
          <el-icon><CircleCheck /></el-icon>
          <span>结果</span>
          <el-button text size="small" @click.stop="copyResult" class="copy-btn">
            <el-icon><CopyDocument /></el-icon>
            复制
          </el-button>
        </div>
        <div class="section-content">
          <pre class="result-content">{{ toolCall.result }}</pre>
        </div>
      </div>
      <!-- 错误显示 -->
      <div v-if="toolCall.error" class="tool-call-section">
        <div class="section-title error-title">
          <el-icon><Warning /></el-icon>
          <span>错误</span>
        </div>
        <div class="section-content error-content">
          <pre>{{ toolCall.error }}</pre>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Check, Close, Loading, ArrowDown, Document, CircleCheck, Warning, CopyDocument } from '@element-plus/icons-vue'
import type { WorkspaceChatToolCallSummary } from '@/api/workspace'

const props = defineProps<{
  toolCall: WorkspaceChatToolCallSummary
}>()

const expanded = ref(true)

function toggleExpand() {
  expanded.value = !expanded.value
}

function formatJSON(jsonStr: string): string {
  try {
    const obj = JSON.parse(jsonStr)
    return JSON.stringify(obj, null, 2)
  } catch {
    return jsonStr
  }
}

async function copyArguments() {
  if (!props.toolCall.arguments) return
  try {
    await navigator.clipboard.writeText(props.toolCall.arguments)
    ElMessage.success('参数已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}

async function copyResult() {
  if (!props.toolCall.result) return
  try {
    await navigator.clipboard.writeText(props.toolCall.result)
    ElMessage.success('结果已复制到剪贴板')
  } catch (e) {
    ElMessage.error('复制失败')
  }
}
</script>

<style scoped lang="scss">
.tool-call-card {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-base);
  background: var(--el-fill-color-lighter);
  margin-bottom: 8px;
  transition: all 0.2s;

  &:hover {
    border-color: var(--el-border-color);
  }

  &--expanded {
    border-color: var(--el-color-primary-light-7);
  }

  &--error {
    border-color: var(--el-color-error-light-7);
  }

  &--streaming {
    border-color: var(--el-color-primary-light-5);
  }
}

.tool-call-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 12px;
  cursor: pointer;
  user-select: none;

  &:hover {
    background: var(--el-fill-color);
  }
}

.tool-call-info {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.tool-call-icon {
  font-size: 16px;

  &.success {
    color: var(--el-color-success);
  }

  &.running,
  &.streaming {
    color: var(--el-color-primary);
  }

  &.error {
    color: var(--el-color-error);
  }
}

.tool-call-name {
  font-weight: 600;
  font-size: 14px;
  color: var(--el-text-color-primary);
}

.tool-call-status {
  margin-left: 4px;
}

.expand-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  transition: transform 0.2s;

  &.expanded {
    transform: rotate(180deg);
  }
}

.tool-call-details {
  border-top: 1px solid var(--el-border-color-lighter);
  padding: 12px;
  background: var(--el-bg-color);
}

.tool-call-section {
  margin-bottom: 12px;

  &:last-child {
    margin-bottom: 0;
  }
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  margin-bottom: 8px;

  .el-icon {
    font-size: 14px;
  }

  &.error-title {
    color: var(--el-color-error);
  }
}

.copy-btn {
  margin-left: auto;
  padding: 0;
  height: auto;
  font-size: 12px;
}

.section-content {
  background: var(--el-fill-color-lighter);
  border: 1px solid var(--el-border-color-lighter);
  border-radius: var(--el-border-radius-small);
  padding: 10px;
  max-height: 300px;
  overflow-y: auto;

  pre {
    margin: 0;
    font-size: 12px;
    line-height: 1.5;
    color: var(--el-text-color-regular);
    white-space: pre-wrap;
    word-break: break-word;
  }

  &.error-content {
    background: var(--el-color-error-light-9);
    border-color: var(--el-color-error-light-7);

    pre {
      color: var(--el-color-error);
    }
  }
}

.json-content {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  color: var(--el-text-color-regular);
}
.json-content--empty {
  color: var(--el-text-color-secondary);
  font-style: italic;
}

.result-content {
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', 'Consolas', monospace;
  color: var(--el-text-color-regular);
}
</style>
