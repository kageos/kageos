<template>
  <div class="mini-prd-confirm-bar" :data-testid="dataTestId">
    <div class="mini-prd-confirm-copy">
      <strong>{{ title }}</strong>
      <span>{{ helpText }}</span>
    </div>
    <div class="mini-prd-confirm-actions">
      <el-button size="small" @click="$emit('view')">{{ viewLabel }}</el-button>
      <el-button size="small" @click="$emit('revise')">{{ reviseLabel }}</el-button>
      <el-button size="small" @click="$emit('cancel')">{{ cancelLabel }}</el-button>
      <el-button type="primary" size="small" :loading="sending" @click="$emit('confirm')">
        {{ confirmLabel }}
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'

const props = defineProps<{
  variant: 'prd' | 'test' | 'repair'
  helpText: string
  sending: boolean
}>()

defineEmits<{
  (e: 'view'): void
  (e: 'revise'): void
  (e: 'cancel'): void
  (e: 'confirm'): void
}>()

const title = computed(() => {
  if (props.variant === 'prd') return 'PRD 等待确认'
  if (props.variant === 'repair') return '构建等待修复'
  return '应用等待测试'
})
const viewLabel = computed(() => {
  if (props.variant === 'prd') return '查看 PRD'
  if (props.variant === 'repair') return '查看诊断'
  return '查看构建结果'
})
const reviseLabel = computed(() => props.variant === 'prd' ? '修改 PRD' : '继续修改')
const cancelLabel = computed(() => {
  if (props.variant === 'prd') return '取消'
  if (props.variant === 'repair') return '暂不修复'
  return '暂不测试'
})
const confirmLabel = computed(() => {
  if (props.variant === 'prd') return '确认 PRD'
  if (props.variant === 'repair') return '交接修复'
  return '开始测试'
})
const dataTestId = computed(() => {
  if (props.variant === 'prd') return 'mini-prd-confirm-bar'
  if (props.variant === 'repair') return 'mini-build-repair-confirm-bar'
  return 'mini-test-confirm-bar'
})
</script>

<style scoped>
.mini-prd-confirm-bar {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  margin: 0 14px;
  padding: 10px 12px;
  border: 1px solid rgba(246, 189, 77, 0.28);
  border-radius: 12px;
  background:
    linear-gradient(90deg, rgba(246, 189, 77, 0.16), rgba(55, 163, 255, 0.08)),
    rgba(8, 13, 24, 0.88);
  box-shadow: 0 -10px 28px rgba(0, 0, 0, 0.18);
}

.mini-prd-confirm-copy {
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 2px;
}

.mini-prd-confirm-copy strong {
  color: #ffe4a3;
  font-size: 13px;
  line-height: 1.2;
}

.mini-prd-confirm-copy span {
  max-width: 560px;
  overflow: hidden;
  color: var(--mini-cyber-muted);
  font-size: 12px;
  line-height: 1.35;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mini-prd-confirm-actions {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  gap: 6px;
}

.mini-prd-confirm-actions :deep(.el-button) {
  margin-left: 0;
}

:global(.mini-ws:not(.mini-ws--maximized)) .mini-prd-confirm-bar {
  align-items: stretch;
  flex-direction: column;
  gap: 8px;
}
</style>
