<!--
  DepartmentSelector - 组织架构选择器组件
  功能：
  - 显示当前选中的组织架构（使用 DepartmentDisplay）
  - 点击后弹出对话框，显示组织架构树供选择
  - 支持清空选择
-->
<template>
  <div class="department-selector">
    <!-- 当前选中的组织架构显示 -->
    <div v-if="selectedDepartmentPath" class="selected-department">
      <DepartmentDisplay
        :full-code-path="selectedDepartmentPath"
        mode="card"
        layout="horizontal"
        size="medium"
      />
      <el-button
        type="danger"
        link
        size="small"
        @click="handleClear"
      >
        清空
      </el-button>
    </div>
    
    <!-- 未选择时显示按钮 -->
    <el-button
      v-else
      :icon="OfficeBuilding"
      @click="dialogVisible = true"
    >
      选择组织架构
    </el-button>
    
    <!-- 组织架构选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="选择组织架构"
      width="500px"
      :close-on-click-modal="false"
    >
      <div class="selector-content">
        <el-tree
          ref="treeRef"
          :data="departmentTree"
          :props="treeProps"
          :default-expand-all="false"
          :highlight-current="true"
          node-key="full_code_path"
          :current-node-key="selectedDepartmentPath || ''"
          @node-click="handleDepartmentSelect"
          class="department-select-tree"
        >
          <template #default="{ node, data }">
            <div class="tree-node">
              <img src="/组织架构.svg" alt="部门" class="node-icon" />
              <span class="node-label">{{ data.name }}</span>
              <span class="node-path">({{ data.full_code_path }})</span>
            </div>
          </template>
        </el-tree>
      </div>
      
      <template #footer>
        <div class="dialog-footer">
          <el-button @click="dialogVisible = false">取消</el-button>
          <el-button
            type="primary"
            :disabled="!tempSelectedPath"
            @click="handleConfirm"
          >
            确定
          </el-button>
        </div>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { ref, computed, watch, nextTick } from 'vue'
import { ElButton, ElDialog, ElTree } from 'element-plus'
import { OfficeBuilding } from '@element-plus/icons-vue'
import type { Department } from '@/api/department'
import { getDepartmentTree } from '@/api/department'
import DepartmentDisplay from '@/architecture/presentation/widgets/DepartmentDisplay.vue'

interface Props {
  modelValue: string | null
  departmentTree?: Department[]
}

const props = withDefaults(defineProps<Props>(), {
  departmentTree: () => []
})

const emit = defineEmits<{
  'update:modelValue': [value: string | null]
}>()

const dialogVisible = ref(false)
const tempSelectedPath = ref<string | null>(null)
const treeRef = ref<InstanceType<typeof ElTree>>()

const treeProps = {
  children: 'children',
  label: 'name'
}

// 部门树数据
const departmentTree = computed(() => {
  return props.departmentTree || []
})

// 当前选中的组织架构路径
const selectedDepartmentPath = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 监听对话框打开，初始化临时选择
watch(dialogVisible, (visible) => {
  if (visible) {
    tempSelectedPath.value = selectedDepartmentPath.value
    // 定位到当前选中的节点
    nextTick(() => {
      if (tempSelectedPath.value && treeRef.value) {
        treeRef.value.setCurrentKey(tempSelectedPath.value)
        // 展开到当前节点
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
        
        const expandedKeys = expandPath(tempSelectedPath.value)
        expandedKeys.forEach(key => {
          const node = treeRef.value?.getNode(key)
          if (node && !node.expanded) {
            node.expand()
          }
        })
      }
    })
  }
})

// 处理组织架构选择
function handleDepartmentSelect(data: Department) {
  tempSelectedPath.value = data.full_code_path
}

// 确认选择
function handleConfirm() {
  selectedDepartmentPath.value = tempSelectedPath.value
  dialogVisible.value = false
}

// 清空选择
function handleClear() {
  selectedDepartmentPath.value = null
}
</script>

<style scoped>
.department-selector {
  width: 100%;
}

.selected-department {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 8px 12px;
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  background-color: var(--el-bg-color);
}

.selector-content {
  max-height: 400px;
  overflow-y: auto;
}

.department-select-tree {
  border: 1px solid var(--el-border-color);
  border-radius: 4px;
  padding: 8px;
}

.department-select-tree :deep(.el-tree-node__content) {
  height: auto;
  padding: 6px 8px;
  margin-bottom: 2px;
}

.department-select-tree :deep(.el-tree-node__content:hover) {
  background-color: var(--el-fill-color);
}

.department-select-tree :deep(.el-tree-node.is-current > .el-tree-node__content) {
  background-color: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  font-weight: 500;
}

.tree-node {
  display: flex;
  align-items: center;
  gap: 8px;
  flex: 1;
}

.node-icon {
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  opacity: 0.8;
}

.node-label {
  font-size: 14px;
  color: var(--el-text-color-primary);
  flex: 1;
}

.node-path {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  font-family: 'Monaco', 'Menlo', 'Ubuntu Mono', monospace;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}
</style>

