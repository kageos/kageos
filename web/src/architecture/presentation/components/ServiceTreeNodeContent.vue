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
  border: none;
  background: #0ea5e9;
  box-shadow: 0 0 0 2px rgba(14, 165, 233, 0.12);
}

.runtime-state-badge-thinking :deep(.el-badge__content) {
  background: #38bdf8;
  box-shadow: 0 0 12px rgba(56, 189, 248, 0.45);
}

.runtime-state-badge-tool :deep(.el-badge__content) {
  background: #f59e0b;
  box-shadow: 0 0 12px rgba(245, 158, 11, 0.42);
}

.runtime-state-badge-approval :deep(.el-badge__content) {
  background: #a855f7;
  box-shadow: 0 0 12px rgba(168, 85, 247, 0.42);
}

.runtime-state-badge-failed :deep(.el-badge__content) {
  background: #ef4444;
  box-shadow: 0 0 12px rgba(239, 68, 68, 0.42);
}

.notification-count-badge {
  flex-shrink: 0;
  margin-left: 6px;
  cursor: pointer;
}

.notification-count-badge :deep(.el-badge__content) {
  border: none;
  background: #ef4444;
  box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.12);
}

.notification-count-badge.is-history :deep(.el-badge__content) {
  background: #94a3b8;
  color: #fff;
  box-shadow: 0 0 0 2px rgba(148, 163, 184, 0.14);
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
