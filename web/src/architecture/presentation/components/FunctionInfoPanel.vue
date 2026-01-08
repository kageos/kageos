<!--
  FunctionInfoPanel - 函数信息面板
  显示函数的详细信息，包括创建者、函数描述、标签、使用说明等
-->
<template>
  <div class="function-info-container">
    <!-- 用户信息头部 -->
    <div class="info-header">
      <div class="user-info-wrapper">
        <UserDisplay 
          v-if="mergedFunctionData.created_by"
          :username="mergedFunctionData.created_by" 
          mode="card"
          layout="horizontal"
          :size="48"
        />
        <div v-else class="user-placeholder">
          <el-avatar :size="48" class="user-avatar-large">
            <el-icon><User /></el-icon>
          </el-avatar>
          <span class="user-name">未知用户</span>
        </div>
      </div>
      <div v-if="mergedFunctionData.created_at" class="create-time">
        <el-icon class="time-icon"><Clock /></el-icon>
        <span>{{ formatDate(mergedFunctionData.created_at) }}</span>
      </div>
    </div>

    <el-divider border-style="double" />

    <!-- 函数详情 -->
    <div class="function-detail">
      <h2>{{ mergedFunctionData.name || '未命名函数' }}</h2>

      <h3 v-if="mergedFunctionData.description">函数简介</h3>
      <el-text v-if="mergedFunctionData.description" size="small" class="mx-1">{{ mergedFunctionData.description }}</el-text>

      <h3 v-if="mergedFunctionData.tags">相关Tag</h3>
      <div v-if="mergedFunctionData.tags" class="function-tags">
        <el-tag 
          v-for="tag in (mergedFunctionData.tags || '').split(',').filter(t => t.trim())" 
          :key="tag"
          effect="plain"
          class="tag-item"
        >
          {{ tag.trim() }}
        </el-tag>
      </div>

      <h3 v-if="hasUsageInfo">使用说明</h3>
      <div v-if="hasUsageInfo" class="function-usage">
        <div class="usage-item" v-if="mergedFunctionData.router">
          <span class="label">接口路径：</span>
          <span class="value">{{ mergedFunctionData.router }}</span>
        </div>
        <div class="usage-item" v-if="mergedFunctionData.method">
          <span class="label">请求方法：</span>
          <span class="value">{{ mergedFunctionData.method }}</span>
        </div>
        <div class="usage-item" v-if="mergedFunctionData.callbacks">
          <span class="label">回调函数：</span>
          <span class="value">{{ mergedFunctionData.callbacks }}</span>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { ElRow, ElCol, ElAvatar, ElText, ElTooltip, ElTag, ElDivider, ElIcon } from 'element-plus'
import { User, Clock } from '@element-plus/icons-vue'
import UserDisplay from '../widgets/UserDisplay.vue'
import type { FunctionDetail } from '../../domain/types'
import type { ServiceTree } from '@/types'

interface Props {
  functionData?: FunctionDetail | null
  functionNode?: ServiceTree | null  // 服务树节点，可能包含额外的元数据
}

const props = defineProps<Props>()

// 合并函数详情和服务树节点的数据
const mergedFunctionData = computed(() => {
  const detail = props.functionData || {}
  const node = props.functionNode || {}
  
  return {
    ...detail,
    // 优先使用 detail 中的数据，如果没有则使用 node 中的数据
    name: detail.name || node.name || '',
    description: detail.description || node.description || '',
    tags: detail.tags || node.tags || '',
    created_by: detail.created_by || node.owner || '',
    created_at: detail.created_at || node.created_at || '',
    router: detail.router || node.full_code_path || '',
    method: detail.method || 'GET',
    callbacks: detail.callbacks || ''
  }
})

// 格式化日期
function formatDate(date: string | Date | undefined): string {
  if (!date) return '-'
  try {
    const d = typeof date === 'string' ? new Date(date) : date
    return d.toLocaleString('zh-CN', {
      year: 'numeric',
      month: '2-digit',
      day: '2-digit',
      hour: '2-digit',
      minute: '2-digit'
    })
  } catch {
    return String(date)
  }
}

// 检查是否有使用说明信息
const hasUsageInfo = computed(() => {
  return !!(mergedFunctionData.value.router || mergedFunctionData.value.method || mergedFunctionData.value.callbacks)
})
</script>

<style lang="scss" scoped>
.function-info-container {
  width: 100%;
  height: 100%;
  background-color: var(--el-bg-color);
  padding: 16px;
  overflow-y: auto;

  .info-header {
    display: flex;
    flex-direction: column;
    gap: 12px;
    padding: 16px;
    background: linear-gradient(135deg, rgba(64, 158, 255, 0.03) 0%, rgba(64, 158, 255, 0.01) 100%);
    border-radius: 8px;
    border: 1px solid var(--el-border-color-lighter);
    margin-bottom: 16px;

    .user-info-wrapper {
      display: flex;
      align-items: center;
    }

    .user-placeholder {
      display: flex;
      align-items: center;
      gap: 12px;

      .user-avatar-large {
        background: linear-gradient(135deg, var(--el-color-primary-light-8), var(--el-color-primary-light-9));
        border: 2px solid var(--el-color-primary-light-5);
      }

      .user-name {
        font-size: 15px;
        font-weight: 500;
        color: var(--el-text-color-primary);
      }
    }

    .create-time {
      display: flex;
      align-items: center;
      gap: 6px;
      font-size: 12px;
      color: var(--el-text-color-secondary);
      padding-left: 4px;

      .time-icon {
        font-size: 14px;
      }
    }
  }

  .el-divider {
    border-color: var(--el-border-color-light);
    margin: 16px 0;
  }

  .function-detail {
    h2 {
      font-size: 18px;
      color: var(--el-text-color-primary);
      margin: 0 0 16px 0;
      font-weight: 500;
    }

    .function-meta {
      color: var(--el-text-color-secondary);
      font-size: 13px;
      margin-bottom: 16px;

      .meta-item {
        display: flex;
        align-items: center;
        gap: 8px;
        white-space: nowrap;
        overflow: hidden;
        text-overflow: ellipsis;

        .label {
          color: var(--el-text-color-secondary);
        }

        .value {
          color: var(--el-text-color-primary);
        }
      }
    }

    h3 {
      font-size: 14px;
      color: var(--el-text-color-primary);
      margin: 16px 0 8px 0;
      font-weight: 500;
    }

    .el-text {
      color: var(--el-text-color-secondary);
      font-size: 13px;
      line-height: 1.5;
    }

    .function-tags {
      display: flex;
      flex-wrap: wrap;
      gap: 8px;
      margin-top: 8px;

      .el-tag {
        background-color: var(--el-fill-color-light);
        border-color: var(--el-border-color);
        color: var(--el-text-color-regular);
        font-size: 12px;
        padding: 0 8px;
        height: 24px;
        line-height: 22px;
        margin: 0;

        &:hover {
          background-color: var(--el-fill-color);
        }
      }
    }

    .function-usage {
      margin-top: 8px;
      background-color: var(--el-fill-color-light);
      border-radius: 8px;
      padding: 12px;

      .usage-item {
        display: flex;
        margin-bottom: 8px;
        font-size: 13px;
        line-height: 1.5;

        &:last-child {
          margin-bottom: 0;
        }

        .label {
          color: var(--el-text-color-secondary);
          width: 80px;
          flex-shrink: 0;
        }

        .value {
          color: var(--el-text-color-primary);
          flex: 1;
          word-break: break-all;
          font-family: 'Monaco', 'Menlo', 'Courier New', monospace;
        }
      }
    }
  }
}
</style>

