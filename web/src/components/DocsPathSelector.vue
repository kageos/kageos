<!--
  DocsPathSelector - 文档路径选择器组件
  功能：
  - 支持多选服务树中的文档路径（package 或 docs 类型节点）
  - 显示已选中的路径
  - 点击后弹出对话框，显示服务树供选择
-->
<template>
  <div class="docs-path-selector">
    <!-- 已选中的路径显示 -->
    <div class="selected-paths">
      <div class="input-wrapper">
        <el-input
          v-model="pathsInput"
          type="textarea"
          :rows="2"
          placeholder="请输入文档路径，多个路径用逗号分隔，如：/system/official/sdk,/system/official/plugins"
          @blur="handleInputBlur"
        />
        <el-button
          :icon="Document"
          @click="handleOpenTreeDialog"
          style="margin-left: 8px;"
        >
          从服务树选择
        </el-button>
      </div>
      <div v-if="selectedPaths.length > 0" class="path-tags" style="margin-top: 8px;">
        <el-tag
          v-for="(path, index) in selectedPaths"
          :key="index"
          closable
          @close="handleRemovePath(index)"
          style="margin-right: 8px; margin-bottom: 4px;"
        >
          {{ path }}
        </el-tag>
      </div>
    </div>
    
    <!-- 文档路径选择对话框 -->
    <el-dialog
      v-model="dialogVisible"
      title="选择文档路径"
      width="600px"
      :close-on-click-modal="false"
    >
      <div class="selector-content">
        <el-input
          v-model="filterText"
          placeholder="搜索节点..."
          clearable
          style="margin-bottom: 12px;"
        >
          <template #prefix>
            <el-icon><Search /></el-icon>
          </template>
        </el-input>
        
        <el-tree
          ref="treeRef"
          :data="filteredTreeData"
          :props="treeProps"
          :default-expand-all="false"
          show-checkbox
          node-key="full_code_path"
          :default-checked-keys="selectedPaths"
          :filter-node-method="filterNode"
          class="docs-path-select-tree"
          @check="handleNodeCheck"
        >
          <template #default="{ node, data }">
            <div class="tree-node">
              <el-icon v-if="data.type === 'docs'" class="node-icon"><Document /></el-icon>
              <el-icon v-else class="node-icon"><Folder /></el-icon>
              <span class="node-label">{{ data.name }}</span>
              <el-tag size="small" :type="getNodeTypeTag(data.type)" style="margin-left: 8px;">
                {{ getNodeTypeLabel(data.type) }}
              </el-tag>
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
import { ref, computed, watch } from 'vue'
import { ElButton, ElDialog, ElTree, ElTag, ElInput, ElIcon, ElMessage } from 'element-plus'
import { Document, Folder, Search } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { getServiceTree } from '@/api/service-tree'

interface Props {
  modelValue: string // 逗号分隔的路径字符串，如："/system/official/sdk,/user/myapp/docs"
  user?: string // 用户（可选，如果不提供则只显示标准库路径）
  app?: string // 应用（可选，如果不提供则只显示标准库路径）
}

const props = withDefaults(defineProps<Props>(), {
  user: '',
  app: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const dialogVisible = ref(false)
const treeRef = ref<InstanceType<typeof ElTree>>()
const filterText = ref('')
const serviceTreeData = ref<ServiceTree[]>([])
const loading = ref(false)
const tempSelectedPaths = ref<string[]>([])
const pathsInput = ref('')

const treeProps = {
  children: 'children',
  label: 'name'
}


// 当前选中的路径数组
const selectedPaths = computed({
  get: () => {
    if (!props.modelValue || typeof props.modelValue !== 'string') return []
    return props.modelValue.split(',').map(p => p.trim()).filter(p => p)
  },
  set: (value: string[]) => {
    emit('update:modelValue', value.length > 0 ? value.join(',') : '')
  }
})

// 同步输入框和选中路径
watch(() => props.modelValue, (newVal) => {
  pathsInput.value = newVal || ''
}, { immediate: true })

// 处理输入框失焦
const handleInputBlur = () => {
  // 从输入框更新 modelValue
  const paths = pathsInput.value.split(',').map(p => p.trim()).filter(p => p)
  selectedPaths.value = paths
}

// 过滤后的树数据（只显示 package 和 docs 类型节点）
const filteredTreeData = computed(() => {
  const filterNode = (nodes: ServiceTree[]): ServiceTree[] => {
    return nodes
      .filter(node => node.type === 'package' || node.type === 'docs')
      .map(node => ({
        ...node,
        children: node.children ? filterNode(node.children) : undefined
      }))
  }
  return filterNode(serviceTreeData.value)
})

// 获取节点类型标签
const getNodeTypeLabel = (type: string) => {
  const typeMap: Record<string, string> = {
    package: '目录',
    docs: '文档'
  }
  return typeMap[type] || type
}

// 获取节点类型标签样式
const getNodeTypeTag = (type: string) => {
  const typeMap: Record<string, string> = {
    package: 'primary',
    docs: 'success'
  }
  return typeMap[type] || 'info'
}

// 过滤节点
const filterNode = (value: string, data: ServiceTree) => {
  if (!value) return true
  return data.name.toLowerCase().includes(value.toLowerCase()) ||
         data.full_code_path.toLowerCase().includes(value.toLowerCase())
}

// 监听过滤文本变化
watch(filterText, (val) => {
  treeRef.value?.filter(val)
})

// 加载服务树数据
const loadServiceTree = async () => {
  if (loading.value) return
  
  loading.value = true
  try {
    const tree = await getServiceTree(props.user!, props.app!)
    serviceTreeData.value = tree || []
  } catch (error: any) {
    console.error('加载服务树失败:', error)
    ElMessage.error(error.message || '加载服务树失败')
    serviceTreeData.value = []
  } finally {
    loading.value = false
  }
}

// 打开对话框
const handleOpenTreeDialog = () => {
  if (!props.user || !props.app) {
    ElMessage.warning('需要指定应用才能从服务树选择路径。请手动输入路径，如：/system/official/sdk')
    return
  }
  dialogVisible.value = true
}

// 打开对话框时加载数据
watch(dialogVisible, (visible) => {
  if (visible) {
    tempSelectedPaths.value = [...selectedPaths.value]
    loadServiceTree()
  }
})

// 处理节点选择
const handleNodeCheck = (data: ServiceTree, checked: { checkedKeys: string[], halfCheckedKeys: string[] }) => {
  tempSelectedPaths.value = checked.checkedKeys
}

// 确认选择
const handleConfirm = () => {
  // 合并已选中的路径和从服务树选择的路径（去重）
  const existingPaths = selectedPaths.value
  const newPaths = tempSelectedPaths.value
  const mergedPaths = Array.from(new Set([...existingPaths, ...newPaths]))
  selectedPaths.value = mergedPaths
  pathsInput.value = mergedPaths.join(',')
  dialogVisible.value = false
}

// 移除路径
const handleRemovePath = (index: number) => {
  const newPaths = [...selectedPaths.value]
  newPaths.splice(index, 1)
  selectedPaths.value = newPaths
  pathsInput.value = newPaths.join(',')
}
</script>

<style scoped lang="scss">
.docs-path-selector {
  .selected-paths {
    width: 100%;
    
    .input-wrapper {
      display: flex;
      align-items: flex-start;
      
      .el-textarea {
        flex: 1;
      }
    }
    
    .path-tags {
      display: flex;
      flex-wrap: wrap;
      align-items: center;
    }
  }
  
  .selector-content {
    max-height: 400px;
    overflow-y: auto;
  }
  
  .docs-path-select-tree {
    .tree-node {
      display: flex;
      align-items: center;
      flex: 1;
      
      .node-icon {
        margin-right: 6px;
        color: var(--el-text-color-secondary);
      }
      
      .node-label {
        font-weight: 500;
      }
      
      .node-path {
        margin-left: 8px;
        font-size: 12px;
        color: var(--el-text-color-secondary);
      }
    }
  }
}
</style>
