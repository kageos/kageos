<!--
  PermissionDeniedView - 权限不足视图组件
  
  职责：
  - 显示权限不足的错误信息
  - 提供申请权限的入口
  - 独立组件，不依赖其他视图
-->

<template>
  <div class="permission-denied-view">
    <el-card class="permission-error-card" shadow="hover">
      <template #header>
        <div class="permission-error-header">
          <el-icon class="permission-error-icon"><Lock /></el-icon>
          <span class="permission-error-title">权限不足</span>
        </div>
      </template>
      <div class="permission-error-content">
        <div class="permission-error-message">
          <p class="error-message-text">
            您没有 <strong>{{ permissionError?.action_display || permissionError?.error_message || '访问该资源' }}</strong> 的权限
          </p>
        </div>
        <div v-if="permissionError?.resource_path" class="permission-error-info">
          <el-icon><Document /></el-icon>
          <span class="info-label">资源路径：</span>
          <span class="info-value">{{ permissionError.resource_path }}</span>
        </div>
        <div v-if="permissionError?.action" class="permission-error-info">
          <el-icon><Key /></el-icon>
          <span class="info-label">权限点：</span>
          <span class="info-value">{{ permissionError.action }}</span>
        </div>
        <div class="permission-error-actions">
          <el-button
            type="primary"
            size="default"
            @click="handleApplyPermission"
            :icon="Lock"
          >
            立即申请权限
          </el-button>
        </div>
      </div>
    </el-card>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useRouter } from 'vue-router'
import { Lock, Document, Key } from '@element-plus/icons-vue'
import { ElCard, ElIcon, ElButton } from 'element-plus'
import { usePermissionErrorStore } from '@/stores/permissionError'
import { buildPermissionApplyURL } from '@/utils/permission'
import type { PermissionInfo } from '@/utils/permission'

const router = useRouter()
const permissionErrorStore = usePermissionErrorStore()

// 获取权限错误信息
const permissionError = computed<PermissionInfo | null>(() => permissionErrorStore.currentError)

// 处理权限申请
const handleApplyPermission = () => {
  if (permissionError.value?.apply_url) {
    if (permissionError.value.apply_url.startsWith('/')) {
      router.push(permissionError.value.apply_url)
    } else {
      window.open(permissionError.value.apply_url, '_blank')
    }
  } else if (permissionError.value?.resource_path && permissionError.value?.action) {
    // 如果没有 apply_url，使用 resource_path 和 action 构建
    const applyURL = buildPermissionApplyURL(permissionError.value.resource_path, permissionError.value.action)
    router.push(applyURL)
  }
}
</script>

<style scoped lang="scss">
.permission-denied-view {
  display: flex;
  justify-content: center;
  align-items: center;
  min-height: 400px;
  padding: 40px 20px;
}

.permission-error-card {
  max-width: 600px;
  width: 100%;
  border-radius: 16px;
  border: none;
  box-shadow: 0 4px 20px rgba(0, 0, 0, 0.08);
  transition: all 0.3s ease;

  &:hover {
    box-shadow: 0 6px 24px rgba(0, 0, 0, 0.12);
    transform: translateY(-2px);
  }
}

.permission-error-header {
  display: flex;
  align-items: center;
  gap: 12px;
  font-size: 18px;
  font-weight: 600;
  color: var(--el-color-warning);
}

.permission-error-icon {
  font-size: 24px;
}

.permission-error-title {
  font-size: 18px;
}

.permission-error-content {
  padding: 8px 0;
}

.permission-error-message {
  margin-bottom: 24px;
  padding: 16px;
  background: linear-gradient(135deg, rgba(255, 193, 7, 0.1) 0%, rgba(255, 152, 0, 0.05) 100%);
  border-radius: 12px;
  border-left: 4px solid var(--el-color-warning);
}

.error-message-text {
  margin: 0;
  font-size: 15px;
  line-height: 1.6;
  color: var(--el-text-color-primary);

  strong {
    color: var(--el-color-warning);
    font-weight: 600;
  }
}

.permission-error-info {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 16px;
  padding: 12px 16px;
  background: var(--el-bg-color-page);
  border-radius: 10px;
  font-size: 14px;
  transition: all 0.2s ease;

  &:hover {
    background: var(--el-fill-color-light);
  }

  .el-icon {
    color: var(--el-color-info);
    font-size: 18px;
  }

  .info-label {
    color: var(--el-text-color-regular);
    font-weight: 500;
  }

  .info-value {
    color: var(--el-text-color-primary);
    font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
    font-size: 13px;
    word-break: break-all;
  }
}

.permission-error-actions {
  margin-top: 24px;
  display: flex;
  justify-content: center;
  padding-top: 16px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>

