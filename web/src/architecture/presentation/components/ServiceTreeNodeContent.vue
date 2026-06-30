<template>
  <span
    class="tree-node"
    :class="{ 'tree-node-draggable': draggable, 'is-active': active }"
    :draggable="draggable"
    :title="title"
  >
    <img
      v-if="node.type === 'package'"
      src="/service-tree/custom-folder.svg"
      :alt="isRootNode(node) ? t('serviceTree.workspaceAlt') : t('serviceTree.directoryAlt')"
      class="node-icon package-icon-img"
      :class="nodeIconClass"
    />
    <template v-else-if="node.type === 'function'">
      <img
        v-if="node.template_type === TEMPLATE_TYPE.FORM"
        src="/service-tree/编辑.svg"
        :alt="t('serviceTree.formAlt')"
        class="node-icon form-icon-img"
        :class="nodeIconClass"
      />
      <el-icon
        v-else
        class="node-icon"
        :class="nodeIconClass"
      >
        <component :is="functionIcon" />
      </el-icon>
    </template>
    <img
      v-else-if="node.type === 'docs'"
      src="/文档.svg"
      :alt="t('serviceTree.docsAlt')"
      class="node-icon docs-icon-img"
      :class="nodeIconClass"
    />
    <span v-else class="node-icon fx-icon" :class="nodeIconClass">fx</span>

    <span class="node-label">{{ displayLabel }}</span>

    <el-badge
      v-if="showRuntimeBadge"
      :value="runtimeBadgeValue"
      :max="99"
      :class="['runtime-state-badge', runtimeBadgeClass]"
      :title="runtimeBadgeTitle"
    />

    <el-badge
      v-if="showScheduledAgentBadge"
      :value="scheduledAgentBadgeValue"
      :max="99"
      class="scheduled-agent-badge"
      :title="scheduledAgentBadgeTitle"
    />

    <el-badge
      v-if="showNotificationBadge"
      :value="notificationBadgeValue"
      :max="99"
      :class="['notification-count-badge', notificationBadgeClass]"
      :title="notificationBadgeTitle"
      @click.stop="$emit('notification-click')"
    />

    <slot name="actions" />
  </span>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import { Document } from '@element-plus/icons-vue'
import ChartIcon from '@/architecture/presentation/shared/components/icons/ChartIcon.vue'
import TableIcon from '@/architecture/presentation/shared/components/icons/TableIcon.vue'
import FormIcon from '@/architecture/presentation/shared/components/icons/FormIcon.vue'
import type { ServiceTree } from '@/architecture/domain/types'
import { TEMPLATE_TYPE } from '@/architecture/domain/constants/functionTypes'
import { isRootNode } from '@/architecture/domain/utils/tree-utils'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  node: ServiceTree
  label?: string
  title?: string
  draggable?: boolean
  active?: boolean
  showRuntimeBadge?: boolean
  runtimeBadgeValue?: string | number
  runtimeBadgeClass?: string
  runtimeBadgeTitle?: string
  showScheduledAgentBadge?: boolean
  scheduledAgentBadgeValue?: string | number
  scheduledAgentBadgeTitle?: string
  showNotificationBadge?: boolean
  notificationBadgeValue?: string | number
  notificationBadgeClass?: string
  notificationBadgeTitle?: string
}>(), {
  label: '',
  title: '',
  draggable: false,
  active: false,
  showRuntimeBadge: false,
  runtimeBadgeValue: '',
  runtimeBadgeClass: '',
  runtimeBadgeTitle: '',
  showScheduledAgentBadge: false,
  scheduledAgentBadgeValue: '',
  scheduledAgentBadgeTitle: '',
  showNotificationBadge: false,
  notificationBadgeValue: '',
  notificationBadgeClass: '',
  notificationBadgeTitle: '',
})

defineEmits<{
  (e: 'notification-click'): void
}>()

const displayLabel = computed(() => {
  return props.label || props.node.name || props.node.code || props.node.full_code_path || '-'
})

const functionIcon = computed(() => {
  if (props.node.template_type === TEMPLATE_TYPE.TABLE) return TableIcon
  if (props.node.template_type === TEMPLATE_TYPE.FORM) return FormIcon
  if (props.node.template_type === TEMPLATE_TYPE.CHART) return ChartIcon
  return Document
})

const nodeIconClass = computed(() => {
  if (props.node.type === 'package') return 'package-icon'
  if (props.node.type === 'function') {
    if (props.node.template_type === TEMPLATE_TYPE.TABLE) return 'table-icon'
    if (props.node.template_type === TEMPLATE_TYPE.FORM) return 'form-icon'
    if (props.node.template_type === TEMPLATE_TYPE.CHART) return 'chart-icon'
    return 'function-icon'
  }
  if (props.node.type === 'docs') return 'docs-icon'
  return 'function-icon'
})
</script>

<style scoped>
.tree-node {
  display: flex;
  width: 100%;
  min-width: 0;
  flex: 1;
  align-items: center;
  gap: 8px;

  &.tree-node-draggable {
    cursor: grab;

    &:active {
      cursor: grabbing;
    }
  }

  &.is-active {
    .node-label {
      color: var(--el-text-color-primary);
      font-weight: 500;
    }

    .node-icon {
      color: #6366f1;
      opacity: 0.9;
    }
  }
}

.node-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  margin-right: 8px;
  color: #6366f1;
  opacity: 0.8;
  transition: color 0.2s ease;

  &.package-icon {
    color: #6366f1;
    opacity: 0.8;
  }

  &.package-icon-img,
  &.form-icon-img,
  &.group-icon-img {
    width: 16px;
    height: 16px;
    object-fit: contain;
    opacity: 0.9;
  }

  &.table-icon {
    color: #10b981;
    opacity: 0.9;
  }

  &.form-icon {
    color: #3b82f6;
    opacity: 0.9;
  }

  &.function-icon {
    color: #6366f1;
    opacity: 0.8;
  }

  &.fx-icon {
    color: #6366f1;
    font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
    font-size: 12px;
    font-style: italic;
    font-weight: 600;
    opacity: 0.8;
  }

  &.group-icon {
    color: #909399;
    opacity: 0.9;
  }
}

.node-label {
  min-width: 0;
  flex: 1;
  overflow: hidden;
  color: var(--el-text-color-primary);
  font-size: 14px;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.runtime-state-badge {
  flex-shrink: 0;
  margin-left: 6px;
}

.runtime-state-badge :deep(.el-badge__content) {
  background: rgba(14, 165, 233, 0.12) !important;
  color: #0ea5e9 !important;
  border: 1px solid rgba(14, 165, 233, 0.25) !important;
  box-shadow: none !important;
  font-weight: 600 !important;
  padding: 0 6px !important;
  border-radius: 12px !important;
}

.runtime-state-badge-thinking :deep(.el-badge__content) {
  background: rgba(14, 165, 233, 0.12) !important;
  color: #0ea5e9 !important;
  border-color: rgba(14, 165, 233, 0.25) !important;
}

.runtime-state-badge-tool :deep(.el-badge__content) {
  background: rgba(245, 158, 11, 0.12) !important;
  color: #d97706 !important;
  border-color: rgba(245, 158, 11, 0.25) !important;
}

.runtime-state-badge-approval :deep(.el-badge__content) {
  background: rgba(168, 85, 247, 0.12) !important;
  color: #a855f7 !important;
  border-color: rgba(168, 85, 247, 0.25) !important;
}

.runtime-state-badge-failed :deep(.el-badge__content) {
  background: rgba(239, 68, 68, 0.12) !important;
  color: #ef4444 !important;
  border-color: rgba(239, 68, 68, 0.25) !important;
}

.scheduled-agent-badge {
  flex-shrink: 0;
  margin-left: 6px;
  cursor: help;
}

.scheduled-agent-badge :deep(.el-badge__content) {
  background: rgba(100, 116, 139, 0.12) !important;
  color: #64748b !important;
  border: 1px solid rgba(100, 116, 139, 0.25) !important;
  box-shadow: none !important;
  font-weight: 600 !important;
  padding: 0 6px !important;
  border-radius: 12px !important;
}

.notification-count-badge {
  flex-shrink: 0;
  margin-left: 6px;
  cursor: pointer;
}

.notification-count-badge :deep(.el-badge__content) {
  background: #ef4444 !important;
  color: #ffffff !important;
  border: none !important;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.1) !important;
  font-weight: 600 !important;
  font-size: 11px !important;
  padding: 0 6px !important;
  height: 16px !important;
  line-height: 16px !important;
  border-radius: 8px !important;
  transition: background-color 0.2s, transform 0.2s !important;
  animation: none !important;
}

.notification-count-badge:hover :deep(.el-badge__content) {
  background: #f87171 !important;
  transform: scale(1.05) !important;
}

.notification-count-badge.is-history :deep(.el-badge__content) {
  background: rgba(148, 163, 184, 0.15) !important;
  color: #94a3b8 !important;
  border: 1px solid rgba(148, 163, 184, 0.2) !important;
  box-shadow: none !important;
}

.notification-count-badge.is-history:hover :deep(.el-badge__content) {
  background: rgba(148, 163, 184, 0.25) !important;
  transform: scale(1.05) !important;
}

:slotted(.node-more-actions) {
  flex-shrink: 0;
  margin-left: auto;
  opacity: 0;
  transition: opacity 0.2s;
}

.tree-node:hover :slotted(.node-more-actions),
.tree-node.is-active :slotted(.node-more-actions) {
  opacity: 1;
}
</style>
