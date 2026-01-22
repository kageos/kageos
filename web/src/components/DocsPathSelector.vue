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
    <div v-if="selectedPaths.length > 0" class="selected-paths">
      <el-tag
        v-for="(path, index) in selectedPaths"
        :key="index"
        closable
        @close="handleRemovePath(index)"
        style="margin-right: 8px; margin-bottom: 8px;"
      >
        {{ path }}
      </el-tag>
      <el-button
        type="primary"
        link
        size="small"
        @click="dialogVisible = true"
        style="margin-left: 8px;"
      >
        添加路径
      </el-button>
    </div>
    
    <!-- 未选择时显示按钮 -->
    <el-button
      v-else
      :icon="Document"
      @click="dialogVisible = true"
    >
      选择文档路径
    </el-button>
    
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
import { ref, computed, watch, nextTick } from 'vue'
import { ElButton, ElDialog, ElTree, ElTag, ElInput, ElIcon, ElMessage } from 'element-plus'
import { Document, Folder, Search } from '@element-plus/icons-vue'
import type { ServiceTree } from '@/types'
import { getServiceTree } from '@/api/service-tree'
import { useAppStore } from '@/stores/app'

interface Props {
  modelValue: string // 逗号分隔的路径字符串，如："/system/official/sdk,/user/myapp/docs"
  user?: string // 用户（可选，如果不提供则使用当前应用的用户）
  app?: string // 应用（可选，如果不提供则使用当前应用）
}

const props = withDefaults(defineProps<Props>(), {
  user: '',
  app: ''
})

const emit = defineEmits<{
  'update:modelValue': [value: string]
}>()

const appStore = useAppStore()
const dialogVisible = ref(false)
const treeRef = ref<InstanceType<typeof ElTree>>()
const filterText = ref('')
const serviceTreeData = ref<ServiceTree[]>([])
const loading = ref(false)
const tempSelectedPaths = ref<string[]>([])

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
    const user = props.user || appStore.currentApp?.user || ''
    const app = props.app || appStore.currentApp?.code || ''
    
    if (!user || !app) {
      ElMessage.warning('请先选择应用')
      return
    }
    
    const tree = await getServiceTree(user, app)
    serviceTreeData.value = tree || []
  } catch (error: any) {
    console.error('加载服务树失败:', error)
    ElMessage.error(error.message || '加载服务树失败')
    serviceTreeData.value = []
  } finally {
    loading.value = false
  }
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
  selectedPaths.value = tempSelectedPaths.value
  dialogVisible.value = false
}

// 移除路径
const handleRemovePath = (index: number) => {
  const newPaths = [...selectedPaths.value]
  newPaths.splice(index, 1)
  selectedPaths.value = newPaths
}
</script>

<style scoped lang="scss">
.docs-path-selector {
  .selected-paths {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
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
