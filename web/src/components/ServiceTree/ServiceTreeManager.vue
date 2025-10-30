<script setup lang="ts">
import { ref, computed, watch, onMounted } from 'vue'
import { ElMessage, ElMessageBox } from 'element-plus'
import { Plus, FolderOpen, Setting } from '@element-plus/icons-vue'
import ServiceTree from './ServiceTree.vue'
import ServiceNodeForm from './ServiceNodeForm.vue'
import { createServiceTree, updateServiceTree, deleteServiceTree } from '@/api/service-tree'
import type { ServiceTree } from '@/types'

interface Props {
  appId: number
  appCode: string
  height?: string
  showToolbar?: boolean
}

interface Emits {
  (e: 'node-select', data: ServiceTree): void
  (e: 'tree-change'): void
}

const props = withDefaults(defineProps<Props>(), {
  height: '600px',
  showToolbar: true
})

const emit = defineEmits<Emits>()

// 数据状态
const treeData = ref<ServiceTree[]>([])
const loading = ref(false)
const currentService = ref<ServiceTree | null>(null)
const expandedKeys = ref<number[]>([])

// 表单状态
const showNodeForm = ref(false)
const formMode = ref<'create' | 'edit'>('create')
const formData = ref<Partial<ServiceTree>>({})
const parentNode = ref<ServiceTree | null>(null)

// 计算属性
const containerStyle = computed(() => ({
  height: props.height
}))

// 刷新树数据
const refreshTree = async () => {
  try {
    loading.value = true
    // 这里应该调用实际的API获取服务树数据
    // 暂时使用模拟数据
    await new Promise(resolve => setTimeout(resolve, 500))

    treeData.value = [
      {
        id: 1,
        name: 'HR管理中心',
        code: 'hr',
        parent_id: 0,
        type: 'package',
        description: '人力资源管理中心，包含招聘、绩效、薪酬等功能模块',
        tags: 'hr,人力资源,管理',
        app_id: props.appId,
        ref_id: 0,
        full_code_path: 'hr',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
        children: [
          {
            id: 2,
            name: '招聘管理',
            code: 'recruitment',
            parent_id: 1,
            type: 'function',
            description: '招聘管理系统，包括职位发布、简历筛选、面试安排等功能',
            tags: '招聘,recruitment,面试',
            app_id: props.appId,
            ref_id: 0,
            full_code_path: 'hr.recruitment',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          },
          {
            id: 3,
            name: '绩效管理',
            code: 'performance',
            parent_id: 1,
            type: 'function',
            description: '绩效管理系统，包括目标设定、绩效评估、结果分析等功能',
            tags: '绩效,performance,评估',
            app_id: props.appId,
            ref_id: 0,
            full_code_path: 'hr.performance',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          },
          {
            id: 4,
            name: '薪酬管理',
            code: 'salary',
            parent_id: 1,
            type: 'function',
            description: '薪酬管理系统，包括薪资计算、发放、统计等功能',
            tags: '薪酬,salary,工资',
            app_id: props.appId,
            ref_id: 0,
            full_code_path: 'hr.salary',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          }
        ]
      },
      {
        id: 5,
        name: '项目管理',
        code: 'project',
        parent_id: 0,
        type: 'package',
        description: '项目管理系统，包括项目规划、任务分配、进度跟踪等功能',
        tags: '项目,管理,任务',
        app_id: props.appId,
        ref_id: 0,
        full_code_path: 'project',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z',
        children: [
          {
            id: 6,
            name: '任务管理',
            code: 'tasks',
            parent_id: 5,
            type: 'function',
            description: '任务管理系统，包括任务创建、分配、跟踪、完成等功能',
            tags: '任务,管理,跟踪',
            app_id: props.appId,
            ref_id: 0,
            full_code_path: 'project.tasks',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          },
          {
            id: 7,
            name: '进度跟踪',
            code: 'progress',
            parent_id: 5,
            type: 'function',
            description: '项目进度跟踪系统，包括里程碑管理、进度报告等功能',
            tags: '进度,跟踪,报告',
            app_id: props.appId,
            ref_id: 0,
            full_code_path: 'project.progress',
            created_at: '2023-01-01T00:00:00Z',
            updated_at: '2023-01-01T00:00:00Z'
          }
        ]
      },
      {
        id: 8,
        name: '财务管理',
        code: 'finance',
        parent_id: 0,
        type: 'package',
        description: '财务管理系统，包括预算、支出、报销等功能',
        tags: '财务,管理,预算',
        app_id: props.appId,
        ref_id: 0,
        full_code_path: 'finance',
        created_at: '2023-01-01T00:00:00Z',
        updated_at: '2023-01-01T00:00:00Z'
      }
    ]

    // 默认展开第一层
    expandedKeys.value = treeData.value.map(item => item.id)

  } catch (error) {
    console.error('获取服务树失败:', error)
    ElMessage.error('获取服务树失败')
  } finally {
    loading.value = false
  }
}

// 查找节点
const findNode = (nodes: ServiceTree[], id: number): ServiceTree | null => {
  for (const node of nodes) {
    if (node.id === id) return node
    if (node.children) {
      const found = findNode(node.children, id)
      if (found) return found
    }
  }
  return null
}

// 生成完整代码路径
const generateFullCodePath = (parentNode: ServiceTree | null, code: string): string => {
  if (!parentNode) return code
  return `${parentNode.full_code_path}.${code}`
}

// 处理节点选择
const handleNodeSelect = (data: ServiceTree) => {
  currentService.value = data
  emit('node-select', data)
}

// 处理创建节点
const handleNodeCreate = async (parentId: number) => {
  const parent = findNode(treeData.value, parentId)
  if (!parent && parentId !== 0) {
    ElMessage.error('找不到父节点')
    return
  }

  if (parent && parent.type !== 'package') {
    ElMessage.error('只能在目录下创建子节点')
    return
  }

  formMode.value = 'create'
  formData.value = {
    parent_id: parentId,
    type: 'package',
    app_id: props.appId
  }
  parentNode.value = parent
  showNodeForm.value = true
}

// 处理编辑节点
const handleNodeEdit = (data: ServiceTree) => {
  formMode.value = 'edit'
  formData.value = { ...data }
  parentNode.value = findNode(treeData.value, data.parent_id)
  showNodeForm.value = true
}

// 处理删除节点
const handleNodeDelete = async (data: ServiceTree) => {
  try {
    await ElMessageBox.confirm(
      `确定要删除"${data.name}"吗？此操作不可恢复。`,
      '确认删除',
      {
        confirmButtonText: '确定',
        cancelButtonText: '取消',
        type: 'warning'
      }
    )

    // 检查是否有子节点
    if (data.children && data.children.length > 0) {
      ElMessage.error('请先删除所有子节点')
      return
    }

    loading.value = true

    // 调用删除API
    // await deleteServiceTree(data.id)

    // 从本地数据中移除
    removeNodeFromTree(treeData.value, data.id)

    ElMessage.success('删除成功')
    emit('tree-change')

  } catch (error) {
    if (error !== 'cancel') {
      console.error('删除节点失败:', error)
      ElMessage.error('删除失败')
    }
  } finally {
    loading.value = false
  }
}

// 从树中移除节点
const removeNodeFromTree = (nodes: ServiceTree[], nodeId: number): boolean => {
  for (let i = 0; i < nodes.length; i++) {
    if (nodes[i].id === nodeId) {
      nodes.splice(i, 1)
      return true
    }
    if (nodes[i].children && removeNodeFromTree(nodes[i].children!, nodeId)) {
      return true
    }
  }
  return false
}

// 处理节点拖拽
const handleNodeDrag = (source: ServiceTree, target: ServiceTree) => {
  // 检查是否可以拖拽到目标节点
  if (target.type !== 'package') {
    ElMessage.error('只能拖拽到目录节点')
    refreshTree()
    return
  }

  // 检查是否会形成循环引用
  if (isDescendant(source, target.id)) {
    ElMessage.error('不能拖拽到自己的子节点')
    refreshTree()
    return
  }

  // 更新父节点ID
  source.parent_id = target.id
  source.full_code_path = generateFullCodePath(target, source.code)

  // 这里应该调用API更新
  ElMessage.success('移动成功')
  emit('tree-change')
}

// 检查是否为目标节点的后代
const isDescendant = (node: ServiceTree, targetId: number): boolean => {
  if (node.id === targetId) return true
  if (node.children) {
    for (const child of node.children) {
      if (isDescendant(child, targetId)) return true
    }
  }
  return false
}

// 处理表单提交
const handleFormSubmit = async (data: Partial<ServiceTree>) => {
  try {
    loading.value = true

    if (formMode.value === 'create') {
      // 生成完整代码路径
      data.full_code_path = generateFullCodePath(parentNode.value, data.code!)
      data.app_id = props.appId

      // 调用创建API
      // const newNode = await createServiceTree(data)

      // 模拟创建
      const newNode: ServiceTree = {
        ...data as ServiceTree,
        id: Date.now(),
        ref_id: 0,
        created_at: new Date().toISOString(),
        updated_at: new Date().toISOString()
      }

      // 添加到本地数据
      addNodeToTree(treeData.value, newNode)

      ElMessage.success('创建成功')

      // 如果创建的是目录，自动展开
      if (newNode.type === 'package') {
        expandedKeys.value.push(newNode.id)
      }

    } else {
      // 更新完整代码路径
      if (formData.value.parent_id !== data.parent_id) {
        const newParent = findNode(treeData.value, data.parent_id!)
        data.full_code_path = generateFullCodePath(newParent, data.code!)
      }

      // 调用更新API
      // await updateServiceTree(data.id!, data)

      // 更新本地数据
      updateNodeInTree(treeData.value, data.id!, data)

      ElMessage.success('更新成功')
    }

    showNodeForm.value = false
    emit('tree-change')

  } catch (error) {
    console.error('保存失败:', error)
    ElMessage.error('保存失败')
  } finally {
    loading.value = false
  }
}

// 添加节点到树
const addNodeToTree = (nodes: ServiceTree[], newNode: ServiceTree): boolean => {
  if (newNode.parent_id === 0) {
    nodes.push(newNode)
    return true
  }

  for (const node of nodes) {
    if (node.id === newNode.parent_id) {
      if (!node.children) node.children = []
      node.children.push(newNode)
      return true
    }
    if (node.children && addNodeToTree(node.children, newNode)) {
      return true
    }
  }
  return false
}

// 更新树中的节点
const updateNodeInTree = (nodes: ServiceTree[], nodeId: number, newData: Partial<ServiceTree>): boolean => {
  for (const node of nodes) {
    if (node.id === nodeId) {
      Object.assign(node, newData)
      return true
    }
    if (node.children && updateNodeInTree(node.children, nodeId, newData)) {
      return true
    }
  }
  return false
}

// 处理工具栏操作
const handleCreateRoot = () => {
  handleNodeCreate(0)
}

// 暴露方法
defineExpose({
  refreshTree,
  getCurrentService: () => currentService.value,
  expandAll: () => {
    // 通过ServiceTree组件的ref调用
  }
})

// 初始化
onMounted(() => {
  refreshTree()
})
</script>

<template>
  <div class="service-tree-manager" :style="containerStyle">
    <!-- 工具栏 -->
    <div class="manager-toolbar" v-if="showToolbar">
      <div class="toolbar-left">
        <h3>{{ appCode }} - 服务目录</h3>
        <span class="node-count">{{ treeData.length }} 个根节点</span>
      </div>
      <div class="toolbar-right">
        <el-button type="primary" :icon="Plus" @click="handleCreateRoot">
          新建根目录
        </el-button>
        <el-button :icon="FolderOpen" @click="refreshTree" :loading="loading">
          刷新
        </el-button>
        <el-button :icon="Setting">
          设置
        </el-button>
      </div>
    </div>

    <!-- 主要内容 -->
    <div class="manager-content">
      <!-- 左侧树形结构 -->
      <div class="tree-panel">
        <ServiceTree
          ref="serviceTreeRef"
          :data="treeData"
          :loading="loading"
          :default-expanded-keys="expandedKeys"
          show-actions
          draggable
          @node-click="handleNodeSelect"
          @node-create="handleNodeCreate"
          @node-edit="handleNodeEdit"
          @node-delete="handleNodeDelete"
          @node-drag="handleNodeDrag"
        >
          <template #toolbar>
            <div class="tree-toolbar-custom">
              <el-input
                placeholder="搜索服务..."
                clearable
                size="small"
                style="width: 200px"
              />
            </div>
          </template>
        </ServiceTree>
      </div>

      <!-- 右侧详情面板 -->
      <div class="detail-panel">
        <slot name="detail" :service="currentService">
          <div v-if="currentService" class="service-detail">
            <el-card>
              <template #header>
                <div class="detail-header">
                  <h4>{{ currentService.name }}</h4>
                  <el-tag :type="currentService.type === 'package' ? 'primary' : 'success'">
                    {{ currentService.type === 'package' ? '目录' : '功能' }}
                  </el-tag>
                </div>
              </template>

              <el-descriptions :column="1" border>
                <el-descriptions-item label="代码">
                  <code>{{ currentService.code }}</code>
                </el-descriptions-item>
                <el-descriptions-item label="完整路径">
                  <code>{{ currentService.full_code_path }}</code>
                </el-descriptions-item>
                <el-descriptions-item label="描述">
                  {{ currentService.description || '暂无描述' }}
                </el-descriptions-item>
                <el-descriptions-item label="标签">
                  <el-tag
                    v-for="tag in (currentService.tags?.split(',') || [])"
                    :key="tag"
                    size="small"
                    class="tag-item"
                  >
                    {{ tag.trim() }}
                  </el-tag>
                </el-descriptions-item>
                <el-descriptions-item label="创建时间">
                  {{ new Date(currentService.created_at).toLocaleString() }}
                </el-descriptions-item>
                <el-descriptions-item label="更新时间">
                  {{ new Date(currentService.updated_at).toLocaleString() }}
                </el-descriptions-item>
              </el-descriptions>

              <div class="detail-actions" v-if="currentService.type === 'function'">
                <el-button type="primary" size="small">
                  打开功能
                </el-button>
                <el-button size="small" @click="handleNodeEdit(currentService)">
                  编辑
                </el-button>
                <el-button type="danger" size="small" @click="handleNodeDelete(currentService)">
                  删除
                </el-button>
              </div>
            </el-card>
          </div>

          <el-empty v-else description="请选择一个服务查看详情" />
        </slot>
      </div>
    </div>

    <!-- 节点表单对话框 -->
    <ServiceNodeForm
      v-model:visible="showNodeForm"
      :mode="formMode"
      :data="formData"
      :parent-node="parentNode"
      @submit="handleFormSubmit"
    />
  </div>
</template>

<style scoped>
.service-tree-manager {
  display: flex;
  flex-direction: column;
  height: 100%;
  background: #fff;
  border-radius: 8px;
  overflow: hidden;
}

.manager-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  border-bottom: 1px solid #e4e7ed;
  background: #fafafa;
}

.toolbar-left {
  display: flex;
  align-items: center;
  gap: 12px;
}

.toolbar-left h3 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.node-count {
  font-size: 12px;
  color: #909399;
  background: #f0f0f0;
  padding: 2px 8px;
  border-radius: 12px;
}

.toolbar-right {
  display: flex;
  gap: 8px;
}

.manager-content {
  flex: 1;
  display: flex;
  overflow: hidden;
}

.tree-panel {
  width: 400px;
  border-right: 1px solid #e4e7ed;
  display: flex;
  flex-direction: column;
}

.detail-panel {
  flex: 1;
  padding: 20px;
  overflow: auto;
}

.tree-toolbar-custom {
  padding: 8px 16px;
  border-bottom: 1px solid #e4e7ed;
}

.service-detail {
  height: 100%;
}

.detail-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.detail-header h4 {
  margin: 0;
  font-size: 16px;
  font-weight: 600;
  color: #303133;
}

.tag-item {
  margin-right: 8px;
  margin-bottom: 4px;
}

.detail-actions {
  margin-top: 20px;
  padding-top: 16px;
  border-top: 1px solid #e4e7ed;
  display: flex;
  gap: 8px;
}

/* 响应式设计 */
@media (max-width: 1200px) {
  .tree-panel {
    width: 350px;
  }
}

@media (max-width: 768px) {
  .manager-content {
    flex-direction: column;
  }

  .tree-panel {
    width: 100%;
    height: 300px;
    border-right: none;
    border-bottom: 1px solid #e4e7ed;
  }

  .manager-toolbar {
    flex-direction: column;
    gap: 12px;
    align-items: stretch;
  }
}
</style>