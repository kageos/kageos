<!--
  DepartmentDetailCard - 组织架构详情卡片（用于 Popover）
  功能：
  - 显示组织架构完整信息（名称、路径、负责人、描述等）
  - 显示组织架构树，并定位到当前组织
  - 提供跳转到组织架构管理页面的按钮
-->
<template>
  <div v-if="departmentInfo" class="department-detail-card">
    <!-- 组织架构基本信息 -->
    <div class="department-header">
      <div class="department-basic-info">
        <div class="department-name-primary">
          {{ departmentInfo.full_name_path || departmentInfo.name }}
        </div>
        <div class="department-path">{{ departmentInfo.full_code_path }}</div>
      </div>
    </div>

    <!-- 组织架构树（定位到当前组织） -->
    <div class="department-tree-section">
      <div class="section-title">
        <el-icon class="section-icon"><OfficeBuilding /></el-icon>
        <span>组织架构树</span>
      </div>
      <div class="tree-container">
        <el-tree
          ref="treeRef"
          :data="treeData"
          :props="treeProps"
          :default-expand-all="false"
          :highlight-current="true"
          node-key="full_code_path"
          :current-node-key="currentPath"
          :expand-on-click-node="false"
          class="department-tree"
        >
          <template #default="{ node, data }">
            <div 
              class="tree-node" 
              :class="{ 'is-current': data.full_code_path === currentPath }"
            >
              <img src="/组织架构.svg" alt="部门" class="node-icon" />
              <span class="node-label">{{ data.name }}</span>
            </div>
          </template>
        </el-tree>
      </div>
    </div>

    <!-- 组织架构详细信息 -->
    <div class="info-section">
      <!-- 负责人 -->
      <div v-if="managersList.length > 0" class="info-item">
        <div class="info-label">
          <el-icon class="info-icon"><UserFilled /></el-icon>
          <span>负责人</span>
        </div>
        <div class="info-value">
          {{ managersList.join('、') }}
        </div>
      </div>

      <!-- 描述 -->
      <div v-if="departmentInfo.description" class="info-item">
        <div class="info-label">
          <el-icon class="info-icon"><Document /></el-icon>
          <span>描述</span>
        </div>
        <div class="info-value">
          {{ departmentInfo.description }}
        </div>
      </div>

      <!-- 状态 -->
      <div class="info-item">
        <div class="info-label">
          <el-icon class="info-icon"><CircleCheck /></el-icon>
          <span>状态</span>
        </div>
        <div class="info-value">
          <el-tag :type="statusTagType" size="small">
            {{ statusText }}
          </el-tag>
        </div>
      </div>
    </div>

    <!-- 操作按钮 -->
    <div class="action-footer">
      <el-button
        type="primary"
        size="small"
        :icon="OfficeBuilding"
        @click="handleGoToOrganizationPage"
      >
        查看完整组织架构
      </el-button>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, ref, watch, nextTick } from 'vue'
import { useRouter } from 'vue-router'
import { ElIcon, ElTag, ElButton, ElTree } from 'element-plus'
import { OfficeBuilding, UserFilled, Document, CircleCheck } from '@element-plus/icons-vue'
import type { Department } from '@/api/department'

interface Props {
  departmentInfo: Department | null
  departmentTree?: Department[]
  currentPath?: string | null
}

const props = withDefaults(defineProps<Props>(), {
  departmentTree: () => [],
  currentPath: null,
})

const router = useRouter()
const treeRef = ref<InstanceType<typeof ElTree>>()

const treeProps = {
  children: 'children',
  label: 'name'
}

// 树数据
const treeData = computed(() => {
  return props.departmentTree || []
})

// 负责人列表
const managersList = computed(() => {
  if (!props.departmentInfo?.managers) {
    return []
  }
  return props.departmentInfo.managers.split(',').map(m => m.trim()).filter(Boolean)
})

// 状态文本和标签类型
const statusText = computed(() => {
  const statusMap: Record<string, string> = {
    'active': '激活',
    'inactive': '停用',
  }
  return statusMap[props.departmentInfo?.status || ''] || props.departmentInfo?.status || '未知'
})

const statusTagType = computed(() => {
  if (props.departmentInfo?.status === 'active') {
    return 'success'
  } else if (props.departmentInfo?.status === 'inactive') {
    return 'danger'
  }
  return 'info'
})

// 当前路径（用于定位）
const currentPath = computed(() => {
  return props.currentPath || props.departmentInfo?.full_code_path || null
})

// 监听 currentPath 变化，定位到当前节点
watch([() => props.currentPath, () => props.departmentInfo], () => {
  if (currentPath.value && treeRef.value) {
    nextTick(() => {
      // 设置当前节点
      treeRef.value?.setCurrentKey(currentPath.value!)
      
      // 展开到当前节点的路径
      const expandPath = (path: string) => {
        const parts = path.split('/').filter(Boolean)
        const expandedKeys: string[] = []
        let currentPath = ''
        for (const part of parts) {
          currentPath = currentPath ? `${currentPath}/${part}` : `/${part}`
          expandedKeys.push(currentPath)
        }
        return expandedKeys
      }
      
      const expandedKeys = expandPath(currentPath.value!)
      // 展开所有父节点
      expandedKeys.forEach(key => {
        const node = treeRef.value?.getNode(key)
        if (node && !node.expanded) {
          node.expand()
        }
      })
    })
  }
}, { immediate: true })

// 跳转到组织架构管理页面
function handleGoToOrganizationPage() {
  router.push('/organization')
}
</script>

<style scoped>
.department-detail-card {
  padding: 16px;
  min-width: 360px;
  max-width: 480px;
}

.department-header {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.department-basic-info {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.department-name-primary {
  font-size: 18px;
  font-weight: 600;
  color: var(--el-text-color-primary);
  line-height: 1.4;
  word-break: break-word;
}

.department-path {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
  line-height: 1.4;
}

.department-tree-section {
  margin-bottom: 16px;
  padding-bottom: 16px;
  border-bottom: 1px solid var(--el-border-color-lighter);
}

.section-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 13px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 12px;
}

.section-icon {
  font-size: 16px;
  color: var(--el-color-primary);
}

.tree-container {
  max-height: 200px;
  overflow-y: auto;
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 4px;
  padding: 8px;
  background: var(--el-fill-color-lighter);
}

.department-tree {
  background: transparent;
}

.department-tree :deep(.el-tree-node__content) {
  height: auto;
  padding: 4px 8px;
  margin-bottom: 2px;
}

.department-tree :deep(.el-tree-node__content:hover) {
  background-color: var(--el-fill-color);
}

.department-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
}

.tree-node.is-current {
  color: var(--el-color-primary);
  font-weight: 500;
}

.node-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  opacity: 0.8;
}

.node-label {
  font-size: 13px;
  color: inherit;
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}

.info-section {
  margin-bottom: 16px;
}

.info-item {
  margin-bottom: 12px;
  
  &:last-of-type {
    margin-bottom: 0;
  }
}

.info-label {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: 12px;
  font-weight: 500;
  color: var(--el-text-color-secondary);
  margin-bottom: 6px;
}

.info-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}

.info-value {
  font-size: 14px;
  color: var(--el-text-color-primary);
  line-height: 1.5;
  padding-left: 20px;
  word-break: break-word;
}

.action-footer {
  display: flex;
  justify-content: center;
  padding-top: 12px;
  border-top: 1px solid var(--el-border-color-lighter);
}
</style>

<style>
/* Popover 全局样式 */
.department-info-popover {
  padding: 0 !important;
  border-radius: 8px;
  box-shadow: 0 4px 16px rgba(0, 0, 0, 0.12);
}
</style>

