<!--
  WorkspaceTabs - 工作空间 Tab 标签页组件
  
  职责：
  - Tab 列表展示
  - Tab 点击处理
  - Tab 编辑处理（添加/删除）
-->

<template>
  <div v-if="tabs.length > 0" class="workspace-tabs-container">
    <div class="workspace-tabs-wrapper">
      <el-tabs
        v-model="activeTabId"
        type="card"
        editable
        class="workspace-tabs"
        @tab-click="handleTabClick"
        @edit="handleTabsEdit"
      >
        <el-tab-pane
          v-for="tab in tabs"
          :key="tab.id"
          :label="tab.title"
          :name="tab.id"
          :closable="tabs.length > 1"
        />
      </el-tabs>
      <!-- 清空所有 Tab 按钮 -->
      <el-button
        v-if="tabs.length > 0"
        type="danger"
        :icon="Close"
        circle
        size="small"
        class="clear-all-tabs-btn"
        title="清空所有标签页"
        @click="handleClearAllClick"
      />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { Close } from '@element-plus/icons-vue'
import { ElMessageBox } from 'element-plus'
import type { WorkspaceTab } from '../../domain/services/WorkspaceDomainService'

interface Props {
  tabs: WorkspaceTab[]
  activeTabId: string
}

interface Emits {
  (e: 'update:activeTabId', value: string): void
  (e: 'tab-click', tab: any): void
  (e: 'tab-edit', targetName: string | undefined, action: 'remove' | 'add'): void
  (e: 'clear-all-tabs'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

const activeTabId = computed({
  get: () => props.activeTabId,
  set: (val) => emit('update:activeTabId', val)
})

const handleTabClick = (tab: any) => {
  emit('tab-click', tab)
}

const handleTabsEdit = (targetName: string | undefined, action: 'remove' | 'add') => {
  emit('tab-edit', targetName, action)
}

const handleClearAllClick = async () => {
  try {
    await ElMessageBox.confirm(
      `确定要清空所有 ${props.tabs.length} 个标签页吗？`,
      '清空所有标签页',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning',
        center: true
      }
    )
    emit('clear-all-tabs')
  } catch {
    // 用户取消，不做任何操作
  }
}
</script>

<style scoped lang="scss">
.workspace-tabs-container {
  background: var(--el-bg-color);
  border-bottom: 1px solid var(--el-border-color-lighter);
  padding: 0 20px;
  box-shadow: 0 1px 2px rgba(0, 0, 0, 0.03);
  position: relative;
}

.workspace-tabs-wrapper {
  display: flex;
  align-items: center;
  gap: 12px;
  position: relative;
}

.clear-all-tabs-btn {
  flex-shrink: 0;
  position: absolute;
  right: 0;
  top: 50%;
  transform: translateY(-50%);
  background: var(--el-bg-color);
  box-shadow: 0 0 8px rgba(0, 0, 0, 0.1);
}

.workspace-tabs {
  flex: 1;
  min-width: 0; // 🔥 关键：允许 flex 子元素缩小，触发横向滚动
  overflow: hidden; // 🔥 隐藏溢出，让 el-tabs 内部处理滚动
}

.workspace-tabs {
  :deep(.el-tabs__header) {
    margin: 0;
    border-bottom: none;
  }

  :deep(.el-tabs__nav-wrap) {
    &::after {
      display: none;
    }
    // 🔥 确保 Tab 栏可以横向滚动
    overflow-x: auto;
    overflow-y: hidden;
    // 🔥 隐藏滚动条（可选，如果需要可以显示）
    // scrollbar-width: thin;
    // &::-webkit-scrollbar {
    //   height: 4px;
    // }
  }

  :deep(.el-tabs__nav) {
    border: none;
    // 🔥 确保 nav 不会换行
    white-space: nowrap;
  }

  :deep(.el-tabs__item) {
    height: 44px;
    line-height: 44px;
    padding: 0 20px;
    margin-right: 8px;
    border: 1px solid var(--el-border-color-lighter);
    border-bottom: none;
    border-radius: 8px 8px 0 0;
    background: var(--el-fill-color-lighter);
    color: var(--el-text-color-regular);
    font-size: 14px;
    font-weight: 500;
    transition: all 0.3s cubic-bezier(0.4, 0, 0.2, 1);
    position: relative;
    overflow: hidden;

    // 添加微妙的阴影
    box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);

    // 悬停效果
    &:hover {
      background: var(--el-fill-color-light);
      color: var(--el-text-color-primary);
      transform: translateY(-1px);
      box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
      border-color: var(--el-border-color);
    }

    // 激活状态
    &.is-active {
      background: var(--el-bg-color);
      color: #6366f1; /* ✅ 与服务目录 fx 图标颜色一致（indigo-500） */
      border-color: var(--el-border-color);
      border-bottom-color: var(--el-bg-color);
      box-shadow: 0 -2px 8px rgba(0, 0, 0, 0.06), 0 2px 4px rgba(0, 0, 0, 0.04);
      transform: translateY(0);

      // 激活状态下的底部指示线
      &::after {
        content: '';
        position: absolute;
        bottom: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: #6366f1; /* ✅ 与服务目录 fx 图标颜色一致 */
        border-radius: 2px 2px 0 0;
      }

      // 激活状态下的背景渐变
      &::before {
        content: '';
        position: absolute;
        top: 0;
        left: 0;
        right: 0;
        height: 2px;
        background: linear-gradient(90deg, 
          rgba(99, 102, 241, 0.3) 0%, 
          #6366f1 50%, 
          rgba(99, 102, 241, 0.3) 100%);
        opacity: 0.3;
      }
    }

    // 关闭按钮样式优化
    :deep(.el-icon-close) {
      width: 16px;
      height: 16px;
      line-height: 16px;
      border-radius: 50%;
      transition: all 0.2s;
      margin-left: 8px;
      font-size: 12px;
      
      &:hover {
        background-color: var(--el-color-danger-light-8);
        color: var(--el-color-danger);
        transform: scale(1.1);
      }
    }

    // 非激活状态的关闭按钮颜色
    &:not(.is-active) :deep(.el-icon-close) {
      color: var(--el-text-color-placeholder);
      
      &:hover {
        background-color: var(--el-fill-color-dark);
        color: var(--el-text-color-primary);
      }
    }
  }

  // 添加按钮样式优化
  :deep(.el-tabs__new-tab) {
    height: 44px;
    line-height: 44px;
    padding: 0 12px;
    margin-left: 8px;
    border: 1px dashed var(--el-border-color-lighter);
    border-radius: 6px;
    background: transparent;
    color: var(--el-text-color-secondary);
    transition: all 0.2s;

    &:hover {
      border-color: #6366f1; /* ✅ 与服务目录 fx 图标颜色一致 */
      color: #6366f1;
      background: rgba(99, 102, 241, 0.08); /* indigo-500 的浅色背景 */
    }
  }
}
</style>


