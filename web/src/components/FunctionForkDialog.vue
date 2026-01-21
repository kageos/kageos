<script setup lang="ts">
/**
 * FunctionForkDialog - 函数组 Fork 对话框（重构版）
 * 
 * 功能：
 * 1. 选择源应用和目标应用
 * 2. 拖拽函数组或目录建立映射关系
 * 3. 可视化显示连接线
 * 4. 批量提交 Fork 操作
 */

import { ref, computed, watch, h, nextTick, onMounted, onUnmounted } from 'vue'
import { useRoute } from 'vue-router'
import { ElDialog, ElButton, ElMessage, ElNotification, ElTag, ElEmpty, ElTree, ElForm, ElFormItem, ElDropdown, ElDropdownMenu, ElDropdownItem, ElInput } from 'element-plus'
import { Delete, ArrowRight, Folder, FolderOpened, Plus, MoreFilled, Loading, InfoFilled } from '@element-plus/icons-vue'
import { getServiceTree, createServiceTree } from '@/api/service-tree'
import { forkFunctionGroup } from '@/api/function'
import { createGroupNode, groupFunctionsByCode, getGroupName } from '@/utils/tree-utils'
import { Logger } from '@/core/utils/logger'
import type { App, ServiceTree as ServiceTreeType, CreateServiceTreeRequest } from '@/types'
import AppSelector from './AppSelector.vue'
import { useAuthStore } from '@/stores/auth'
import UsersWidget from '@/architecture/presentation/widgets/UsersWidget.vue'
import type { FieldConfig, FieldValue } from '@/architecture/domain/types'
import { WidgetType } from '@/core/constants/widget'

// 导入工具类
import { MappingManager, type ForkMapping } from '@/utils/fork-dialog/MappingManager'
import { TreeTransformer } from '@/utils/fork-dialog/TreeTransformer'
import { DragHandler } from '@/utils/fork-dialog/DragHandler'
import { ConnectionLineManager, type ConnectionLine } from '@/utils/fork-dialog/ConnectionLineManager'

// Props 和 Emits
interface Props {
  modelValue: boolean
  sourceFullGroupCode?: string
  sourceGroupName?: string
  sourceApp?: App
  currentApp?: App
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()
const route = useRoute()
const authStore = useAuthStore()

// 🔥 判断是否在新版本路由（统一使用 /workspace）
const isV2Route = computed(() => {
  return route.path.startsWith('/workspace')
})

// 对话框显示状态
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 解析源应用信息
const parseSourceApp = (fullGroupCode?: string): { user: string; app: string } | null => {
  if (!fullGroupCode) return null
  const parts = fullGroupCode.split('/').filter(Boolean)
  if (parts.length >= 2) {
    return { user: parts[0], app: parts[1] }
  }
  return null
}

const sourceAppInfo = computed(() => {
  if (props.sourceApp) {
    return { user: props.sourceApp.user, app: props.sourceApp.code, appObj: props.sourceApp }
  }
  if (props.sourceFullGroupCode) {
    const parsed = parseSourceApp(props.sourceFullGroupCode)
    if (parsed) return { ...parsed, appObj: null }
      }
  if (props.currentApp) {
    return { user: props.currentApp.user, app: props.currentApp.code, appObj: props.currentApp }
  }
  return null
})

// 源函数组代码（用于高亮显示）
const sourceFullGroupCode = computed(() => props.sourceFullGroupCode)

// 应用选择器
const showAppSelector = ref(false)
const selectedApp = ref<App | null>(null)

// 服务目录树
const sourceServiceTree = ref<ServiceTreeType[]>([])
const loadingSourceTree = ref(false)
const targetServiceTree = ref<ServiceTreeType[]>([])
const loadingTargetTree = ref(false)

// DOM 引用
const sourceTreeRef = ref<HTMLElement | null>(null)
const targetTreeRef = ref<HTMLElement | null>(null)
const forkLayoutRef = ref<HTMLElement | null>(null)

// 工具类实例
const mappingManager = new MappingManager([])
const treeTransformer = new TreeTransformer(mappingManager)
const dragHandler = new DragHandler(mappingManager, null)
const connectionLineManager = new ConnectionLineManager(
  mappingManager,
  sourceTreeRef,
  targetTreeRef,
  forkLayoutRef
)

// 拖拽状态
const draggedNode = ref<ServiceTreeType | null>(null)
const dragOverNode = ref<ServiceTreeType | null>(null)
const isDragging = ref(false)

// 连接线
const connectionLines = ref<ConnectionLine[]>([])

// 创建目录对话框
const createDirectoryDialogVisible = ref(false)

// 获取当前用户名作为默认管理员
const getDefaultAdmins = () => {
  return authStore.user?.username || ''
}

const createDirectoryForm = ref<CreateServiceTreeRequest>({
  user: '',
  app: '',
  name: '',
  code: '',
  parent_id: 0,
  description: '',
  tags: '',
  admins: getDefaultAdmins()  // 默认当前用户为管理员
})
const creatingDirectory = ref(false)
const currentParentNode = ref<ServiceTreeType | null>(null)

// 管理员字段配置（用于 UsersWidget）
const adminsField = computed<FieldConfig>(() => ({
  code: 'admins',
  name: '管理员',
  widget: {
    type: WidgetType.USERS,
    config: {}
  }
}))

// 管理员字段值（用于 UsersWidget）
const adminsFieldValue = computed<FieldValue>(() => {
  if (!createDirectoryForm.value.admins || !createDirectoryForm.value.admins.trim()) {
    return {
      raw: null,
      display: '',
      meta: {}
    }
  }
  
  const admins = createDirectoryForm.value.admins.split(',').map(s => s.trim()).filter(s => s)
  return {
    raw: admins.join(','),
    display: admins.join(', '),
    meta: {}
  }
})

// 处理管理员字段变化
function handleAdminsChange(value: FieldValue) {
  createDirectoryForm.value.admins = value.raw || ''
}

// 加载应用列表
// 处理应用选择
const handleAppSelect = (app: App) => {
  selectedApp.value = app
  dragHandler.setTargetApp(app)
  
  if (app) {
    loadTargetServiceTree(app).then(() => {
      nextTick().then(() => {
        connectionLineManager.updateLines().then(() => {
          connectionLines.value = connectionLineManager.getLines()
        })
      })
    })
  } else {
    targetServiceTree.value = []
    connectionLines.value = []
  }
  
  Logger.info('FunctionForkDialog', '应用已选择', { app: app?.name, appCode: app?.code })
}

// 加载源服务目录树
const loadSourceServiceTree = async () => {
  if (!sourceAppInfo.value) return
  
  try {
    loadingSourceTree.value = true
    const tree = await getServiceTree(sourceAppInfo.value.user, sourceAppInfo.value.app)
    sourceServiceTree.value = tree || []
    mappingManager.setSourceTree(sourceServiceTree.value)
  } catch (error) {
    Logger.error('FunctionForkDialog', '加载源服务目录树失败', error)
    ElMessage.error('加载源服务目录树失败')
    sourceServiceTree.value = []
  } finally {
    loadingSourceTree.value = false
  }
}

// 加载目标服务目录树
const loadTargetServiceTree = async (app: App) => {
  try {
    loadingTargetTree.value = true
    const tree = await getServiceTree(app.user, app.code)
    targetServiceTree.value = tree || []
  } catch (error) {
    Logger.error('FunctionForkDialog', '加载目标服务目录树失败', error)
    ElMessage.error('加载目标服务目录树失败')
    targetServiceTree.value = []
  } finally {
    loadingTargetTree.value = false
  }
}


// 映射关系列表（响应式）
const mappings = ref<ForkMapping[]>([])

// 更新映射列表（从 MappingManager 同步）
const updateMappings = () => {
  const newMappings = mappingManager.getAllMappings()
  mappings.value = newMappings
}

// 转换后的树结构
// 注意：这些 computed 依赖于 mappings，这样当映射变化时会自动重新计算
const groupedSourceTree = computed(() => {
  return treeTransformer.transformSourceTree(sourceServiceTree.value)
})

const groupedTargetTree = computed(() => {
  if (!selectedApp.value) return []
  const rootPath = `/${selectedApp.value.user}/${selectedApp.value.code}`
  return treeTransformer.transformTargetTree(targetServiceTree.value, rootPath)
})

// 拖拽开始
const handleDragStart = (node: ServiceTreeType, event: DragEvent) => {
  const isGroup = (node as any).isGroup && node.full_group_code
  const isPackage = node.type === 'package' && !(node as any).isGroup
  
  if (!isGroup && !isPackage) {
    event.preventDefault()
    return false
  }
  
  // 验证是否可以拖拽
  const canDrag = dragHandler.canDrag(node)
  if (!canDrag.allowed) {
    if (canDrag.reason) {
      ElMessage.warning(canDrag.reason)
    }
      event.preventDefault()
      return false
  }
  
  draggedNode.value = node
  isDragging.value = true
  
    const data = isGroup ? node.full_group_code : `__package__${node.id}`
  event.dataTransfer?.setData('text/plain', data)
}

// 拖拽结束
const handleDragEnd = () => {
  draggedNode.value = null
  dragOverNode.value = null
  isDragging.value = false
  
  // 拖动结束后，更新连接线
  nextTick(() => {
    connectionLineManager.updateLines().then(() => {
      const lines = connectionLineManager.getLines()
      connectionLines.value = lines
    })
  })
}

// 拖拽悬停
const handleDragOver = (node: ServiceTreeType | null, event: DragEvent) => {
  if (node !== null && (node.type !== 'package' || (node as any).isGroup)) {
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'none'
    }
    return
  }
  
  event.preventDefault()
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  dragOverNode.value = node
}

// 拖拽离开
const handleDragLeave = () => {
  dragOverNode.value = null
}

// 拖拽放置
const handleDrop = async (node: ServiceTreeType | null, event: DragEvent) => {
  event.preventDefault()
  
  if (!draggedNode.value || !selectedApp.value) {
    dragOverNode.value = null
    draggedNode.value = null
    return
  }
  
  const sourceNode = draggedNode.value
  const isSourceGroup = (sourceNode as any).isGroup && sourceNode.full_group_code
  const isSourcePackage = sourceNode.type === 'package' && !(sourceNode as any).isGroup
  
  if (isSourceGroup) {
    // 处理函数组拖拽
    const result = dragHandler.handleGroupDrag(sourceNode, node)
    if (result.success) {
      updateMappings() // 更新响应式映射列表
      ElMessage.success(result.message || '已添加映射关系')
      await nextTick()
      await loadTargetServiceTree(selectedApp.value)
      // 等待 DOM 完全更新（目录拖拽需要更长时间，因为需要等待新目录渲染）
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 500))
      await connectionLineManager.updateLines()
      const lines = connectionLineManager.getLines()
      connectionLines.value = lines
    } else if (result.message) {
      ElMessage.warning(result.message)
    }
  } else if (isSourcePackage) {
    // 处理目录拖拽
    const result = await dragHandler.handleDirectoryDrag(sourceNode, node)
    if (result.success) {
      updateMappings() // 更新响应式映射列表
      ElMessage.success(result.message || '已添加映射关系')
      await nextTick()
      await loadTargetServiceTree(selectedApp.value)
      // 等待 DOM 完全更新（目录拖拽需要更长时间，因为需要等待新目录渲染）
      await nextTick()
      await new Promise(resolve => setTimeout(resolve, 500))
      await connectionLineManager.updateLines()
      const lines = connectionLineManager.getLines()
      connectionLines.value = lines
    } else if (result.message) {
      ElMessage.error(result.message)
    }
  }

      dragOverNode.value = null
      draggedNode.value = null
    }
    
// 处理节点展开/折叠（更新连接线位置）
const handleNodeExpand = () => {
  // 延迟更新，等待展开动画完成
    nextTick(() => {
      setTimeout(() => {
      if (connectionLines.value.length > 0) {
        connectionLineManager.updateLines().then(() => {
          const lines = connectionLineManager.getLines()
          connectionLines.value = lines
      })
      }
    }, 200) // 等待展开动画完成（Element Plus 默认动画时长约 300ms）
  })
      }

const handleNodeCollapse = () => {
  // 延迟更新，等待折叠动画完成
  nextTick(() => {
    setTimeout(() => {
      if (connectionLines.value.length > 0) {
        connectionLineManager.updateLines().then(() => {
          const lines = connectionLineManager.getLines()
          // 过滤掉无效的连接线（元素不可见时会被过滤）
          connectionLines.value = lines
        })
      }
    }, 300) // 等待折叠动画完成（增加延迟确保 DOM 完全更新）
  })
}

// 删除映射
const removeMapping = (index: number) => {
  mappingManager.removeMappingByIndex(index)
  updateMappings() // 更新响应式映射列表
        nextTick(() => {
    connectionLineManager.updateLines().then(() => {
      const lines = connectionLineManager.getLines()
      connectionLines.value = lines
    })
  })
}

// 创建目录
const handleCreateDirectory = (parentNode?: ServiceTreeType) => {
  if (!selectedApp.value) {
    ElMessage.warning('请先选择目标应用')
    return
  }
  
  currentParentNode.value = parentNode || null
  createDirectoryForm.value = {
    user: selectedApp.value.user,
    app: selectedApp.value.code,
    name: '',
    code: '',
    parent_id: parentNode ? Number(parentNode.id) : 0,
    description: '',
    tags: '',
    admins: getDefaultAdmins()  // 默认当前用户为管理员
  }
  createDirectoryDialogVisible.value = true
}

const handleSubmitCreateDirectory = async () => {
  if (!selectedApp.value) {
    ElMessage.warning('请先选择目标应用')
    return
  }
  
  if (!createDirectoryForm.value.name || !createDirectoryForm.value.code) {
    ElMessage.warning('请输入目录名称和代码')
    return
  }
  
  try {
    creatingDirectory.value = true
    await createServiceTree(createDirectoryForm.value)
    ElMessage.success('创建目录成功')
    
    createDirectoryDialogVisible.value = false
    currentParentNode.value = null
    
    await loadTargetServiceTree(selectedApp.value)
  } catch (error: any) {
    Logger.error('FunctionForkDialog', '创建目录失败', error)
    ElMessage.error(error?.message || '创建目录失败')
  } finally {
    creatingDirectory.value = false
  }
}

// 提交 Fork
const handleSubmit = async () => {
  if (!selectedApp.value) {
    ElMessage.warning('请选择目标应用')
    return
  }
  
  if (mappings.value.length === 0) {
    ElMessage.warning('请至少添加一个映射关系')
    return
  }
  
  dialogVisible.value = false
  
  try {
    // 只提交函数组映射，过滤掉目录映射
    const sourceToTargetMap: Record<string, string> = {}
    const functionGroupMappings: ForkMapping[] = []
    
    mappings.value.forEach(mapping => {
      // 判断是否是函数组映射（source 必须是函数组的 full_group_code）
      if (mappingManager.isFunctionGroupMapping(mapping.source)) {
      sourceToTargetMap[mapping.source] = mapping.target
        functionGroupMappings.push(mapping)
      }
    })
    
    if (Object.keys(sourceToTargetMap).length === 0) {
      ElMessage.warning('没有可提交的函数组映射')
      return
    }
    
    const targetApp = { ...selectedApp.value }
    const savedMappings = [...functionGroupMappings]
    
    // 显示"克隆中"的通知
    let notification: any = null
    notification = ElNotification({
      title: '闪电克隆中',
      message: h('div', { 
        class: 'fork-notification-content',
        style: 'line-height: 1.6;' 
      }, [
        h('p', { 
          class: 'fork-notification-text',
          style: 'margin: 0 0 8px 0; font-size: 14px; font-weight: 500;' 
        }, `正在克隆 ${savedMappings.length} 个函数组...`),
        h('p', { 
          class: 'fork-notification-text',
          style: 'margin: 0 0 12px 0; font-size: 13px; line-height: 1.5;' 
        }, '克隆操作正在后台执行，请稍候'),
        h('div', { 
          style: 'margin-top: 8px; display: flex; align-items: center;' 
        }, [
          h('el-icon', { 
            style: 'animation: spin 1s linear infinite; display: inline-block; margin-right: 8px; color: #409EFF;' 
          }, () => h(Loading)),
          h('span', { 
            class: 'fork-notification-text',
            style: 'font-size: 13px; font-weight: 500;' 
          }, '处理中...')
        ])
      ]),
      type: 'info',
      duration: 0, // 不自动关闭
      position: 'top-right',
      showClose: false,
      customClass: 'fork-progress-notification'
    })
    
    await forkFunctionGroup({
      source_to_target_map: sourceToTargetMap,
      target_app_id: selectedApp.value.id
    })
    
    // 关闭"克隆中"的通知
    notification.close()
    
    if (targetApp && targetApp.user && targetApp.code) {
      // 显示"克隆成功"的通知，包含跳转按钮
      // 使用统一的工作空间路径
      const basePath = '/workspace'
      
      ElNotification({
        title: '克隆成功',
        message: h('div', { 
          style: 'line-height: 1.6; color: #303133; background: transparent;' 
        }, [
          h('p', { 
            style: 'margin: 0 0 8px 0; color: #303133; font-size: 14px; font-weight: 500;' 
          }, `成功提交 ${savedMappings.length} 个函数组的克隆任务`),
          h('p', { 
            style: 'margin: 0 0 12px 0; color: #606266; font-size: 13px; line-height: 1.5;' 
          }, '克隆操作正在后台执行，完成后即可使用'),
          h(ElButton, {
            type: 'primary',
            size: 'small',
            style: 'margin-top: 4px;',
            onClick: () => {
              const forkedPaths = savedMappings.map((m: ForkMapping) => m.target).join(',')
              const url = `${basePath}/${targetApp.user}/${targetApp.code}${forkedPaths ? `?_forked=${encodeURIComponent(forkedPaths)}` : ''}`
              // 🔥 如果在新版本路由，在当前窗口跳转；否则在新窗口打开
              if (isV2Route.value) {
                window.location.href = url
              } else {
                window.open(url, '_blank')
              }
            }
          }, () => `跳转到 ${targetApp.name || targetApp.code}`)
        ]),
        type: 'success',
        duration: 0, // 不自动关闭，让用户点击跳转
        position: 'top-right',
        customClass: 'fork-success-notification'
      })
    }
    
    // 提交成功后清除映射关系和连接线
    mappingManager.clear()
    connectionLines.value = []
    updateMappings()
    
    emit('success')
  } catch (error: any) {
    Logger.error('FunctionForkDialog', 'Fork 失败', error)
    // 如果通知存在，关闭它
    if (notification) {
      notification.close()
    }
    ElNotification({
      title: '克隆失败',
      message: error?.message || '克隆操作失败，请重试',
      type: 'error',
      duration: 5000,
      position: 'top-right'
    })
    dialogVisible.value = true
  }
}

// 取消
const handleCancel = () => {
  dialogVisible.value = false
}

// 重置表单
const resetForm = () => {
  mappingManager.clear()
  updateMappings() // 更新响应式映射列表
  selectedApp.value = null
  dragHandler.setTargetApp(null)
  targetServiceTree.value = []
  connectionLines.value = []
}

// 格式化路径（显示用）
const formatPath = (path: string): string => {
  const parts = path.split('/').filter(Boolean)
  if (parts.length <= 2) return path
  return parts.slice(2).join('/')
}

// 获取连接线路径
const getConnectionPath = (sourceRect: DOMRect, targetRect: DOMRect): string => {
  if (!forkLayoutRef.value) return ''
  
  const layoutRect = forkLayoutRef.value.getBoundingClientRect()
  const sourceX = sourceRect.left + sourceRect.width / 2 - layoutRect.left
  const sourceY = sourceRect.top + sourceRect.height / 2 - layoutRect.top
  const targetX = targetRect.left + targetRect.width / 2 - layoutRect.left
  const targetY = targetRect.top + targetRect.height / 2 - layoutRect.top
  
  const controlPoint1X = sourceX + (targetX - sourceX) * 0.5
  const controlPoint1Y = sourceY
  const controlPoint2X = sourceX + (targetX - sourceX) * 0.5
  const controlPoint2Y = targetY
  
  return `M ${sourceX} ${sourceY} C ${controlPoint1X} ${controlPoint1Y}, ${controlPoint2X} ${controlPoint2Y}, ${targetX} ${targetY}`
}

// 监听映射变化，更新连接线和树结构
watch(mappings, () => {
  // 如果正在拖动，不更新连接线（避免拖动过程中的抖动）
  if (isDragging.value) {
    return
  }

  nextTick(() => {
    connectionLineManager.updateLines().then(() => {
      const lines = connectionLineManager.getLines()
      connectionLines.value = lines
    })
  })
}, { deep: true })

// 监听对话框打开
watch(dialogVisible, (visible: boolean) => {
  if (visible) {
    loadSourceServiceTree()
    resetForm()
  }
})

// 生命周期
onMounted(() => {
  if (dialogVisible.value) {
    loadSourceServiceTree()
  }
})

onUnmounted(() => {
  resetForm()
})
</script>

<template>
  <ElDialog
    v-model="dialogVisible"
    title="克隆函数组"
    width="1200px"
    :close-on-click-modal="false"
    align-center
    @close="resetForm"
    class="fork-dialog"
  >
    <div class="fork-dialog-content">
      <!-- 顶部：选择目标应用 -->
      <div class="target-app-selector">
        <ElButton
          type="primary"
          :icon="selectedApp ? FolderOpened : Plus"
          @click="showAppSelector = true"
          style="width: 100%"
        >
          {{ selectedApp ? `${selectedApp.name} (${selectedApp.code})` : '选择目标应用' }}
        </ElButton>
      </div>
      
      <!-- 应用选择器弹窗 -->
      <AppSelector
        v-model="showAppSelector"
        @select="handleAppSelect"
      />

      <!-- 左右分栏布局 -->
      <div class="fork-layout" v-if="sourceAppInfo" ref="forkLayoutRef">
        <!-- SVG 连接线层 -->
        <svg
          v-if="connectionLines.length > 0 && forkLayoutRef"
          class="connection-lines-layer"
          :width="forkLayoutRef?.clientWidth || 1200"
          :height="forkLayoutRef?.clientHeight || 600"
        >
          <defs>
            <marker
              v-for="(line, index) in connectionLines"
              :key="`arrow-${index}`"
              :id="`arrowhead-${index}`"
              markerWidth="10"
              markerHeight="10"
              refX="9"
              refY="3"
              orient="auto"
            >
              <polygon :points="`0 0, 10 3, 0 6`" :fill="line.color.border" />
            </marker>
          </defs>
          <path
            v-for="(line, index) in connectionLines"
            :key="index"
            :d="line.sourceRect && line.targetRect ? getConnectionPath(line.sourceRect, line.targetRect) : ''"
            :stroke="line.color.border"
            :stroke-width="2"
            fill="none"
            :marker-end="`url(#arrowhead-${index})`"
            :opacity="0.7"
            class="connection-line"
            style="stroke-dasharray: 8, 4;"
          />
        </svg>
        
        <!-- 左侧：源应用的服务目录树 -->
        <div class="source-panel">
          <div class="panel-header">
            <h3>源应用：{{ sourceAppInfo.user }}/{{ sourceAppInfo.app }}</h3>
            <ElTag type="info" size="small">拖拽函数组到右侧</ElTag>
          </div>
          <div class="panel-content" v-loading="loadingSourceTree" ref="sourceTreeRef">
            <el-tree
              v-if="groupedSourceTree.length > 0"
              :data="groupedSourceTree"
              :props="{ children: 'children', label: 'name' }"
              :expand-on-click-node="false"
              default-expand-all
              class="source-tree"
              @node-expand="handleNodeExpand"
              @node-collapse="handleNodeCollapse"
            >
              <template #default="{ node, data }">
                <div
                  :data-node-id="(data as any).isGroup && data.full_group_code ? data.full_group_code : (data.id || data.full_code_path)"
                  class="tree-node-wrapper"
                  :class="{
                    'is-draggable': ((data as any).isGroup && data.full_group_code && data.full_group_code.trim() !== '') || (data.type === 'package' && !(data as any).isGroup),
                    'is-dragging': draggedNode?.id === data.id,
                    'is-group': (data as any).isGroup,
                    'is-package': data.type === 'package' && !(data as any).isGroup,
                    'has-mapping': !!(data as any).mappingColor
                  }"
                  :style="(data as any).mappingColor ? {
                    background: (data as any).mappingColor.bg,
                    border: `2px solid ${(data as any).mappingColor.border}`,
                    color: (data as any).mappingColor.text,
                    fontWeight: '500'
                  } : {}"
                  :draggable="((data as any).isGroup && data.full_group_code && data.full_group_code.trim() !== '') || (data.type === 'package' && !(data as any).isGroup)"
                  @dragstart="((data as any).isGroup && data.full_group_code) || (data.type === 'package' && !(data as any).isGroup) ? handleDragStart(data, $event) : null"
                  @dragend="handleDragEnd"
                >
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup" class="node-icon package-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup" class="node-icon group-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <FolderOpened />
                  </el-icon>
                  <span v-else class="node-icon fx-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">fx</span>
                  <span class="node-label" :class="{ 'group-label': (data as any).isGroup }" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup && !(data as any).mappingColor" type="info" size="small" class="group-tag" style="margin-left: 8px;">
                    组
                  </ElTag>
                  <ElTag v-if="data.full_group_code === sourceFullGroupCode" type="warning" size="small" style="margin-left: 8px;">
                    当前
                  </ElTag>
                </div>
              </template>
            </el-tree>
            <ElEmpty v-else-if="!loadingSourceTree" description="暂无服务目录" :image-size="80" />
          </div>
        </div>

        <!-- 中间：箭头提示 -->
        <div class="arrow-panel">
          <el-icon class="arrow-icon"><ArrowRight /></el-icon>
          <div class="arrow-text">拖拽到目标目录</div>
        </div>

        <!-- 右侧：目标应用的服务目录树 -->
        <div class="target-panel" v-if="selectedApp">
          <div class="panel-header">
            <h3>目标应用：{{ selectedApp.user }}/{{ selectedApp.code }}</h3>
            <ElTag type="success" size="small">拖拽到这里</ElTag>
          </div>
          <div class="panel-content" v-loading="loadingTargetTree" ref="targetTreeRef">
            <!-- 根目录拖拽区域 -->
            <div
              data-node-id="root"
              class="root-drop-zone"
              :class="{ 'is-drag-over': dragOverNode === null && isDragging }"
              @dragover="handleDragOver(null, $event)"
              @dragleave="handleDragLeave"
              @drop="handleDrop(null, $event)"
            >
              <el-icon class="root-icon"><Folder /></el-icon>
              <span class="root-label">根目录</span>
              <ElTag type="success" size="small" style="margin-left: 8px;">拖拽到这里</ElTag>
            </div>
            
            <el-tree
              v-if="groupedTargetTree.length > 0"
              :data="groupedTargetTree"
              :props="{ children: 'children', label: 'name' }"
              :expand-on-click-node="false"
              default-expand-all
              class="target-tree"
              @node-expand="handleNodeExpand"
              @node-collapse="handleNodeCollapse"
            >
              <template #default="{ node, data }">
                <div
                  :data-node-id="(data as any).isGroup && data.full_group_code ? data.full_group_code : ((data as any).isPending && data.full_group_code ? data.full_group_code : (data.id || data.full_code_path))"
                  class="tree-node-wrapper"
                  :class="{
                    'is-drag-over': dragOverNode?.id === data.id,
                    'is-package': data.type === 'package' && !(data as any).isGroup && !(data as any).isPending,
                    'is-group': (data as any).isGroup && !(data as any).isPending,
                    'is-pending': (data as any).isPending,
                    'has-mapping': !!(data as any).mappingColor
                  }"
                  :style="(data as any).mappingColor ? {
                    background: (data as any).mappingColor.bg,
                    border: `2px solid ${(data as any).mappingColor.border}`,
                    color: (data as any).mappingColor.text,
                    fontWeight: '500',
                    boxShadow: `0 2px 8px ${(data as any).mappingColor.border}40`
                  } : {}"
                  @dragover="data.type === 'package' && !(data as any).isGroup ? handleDragOver(data, $event) : null"
                  @dragleave="handleDragLeave"
                  @drop="data.type === 'package' && !(data as any).isGroup ? handleDrop(data, $event) : null"
                >
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup && !(data as any).isPending" class="node-icon package-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup && !(data as any).isPending" class="node-icon group-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <FolderOpened />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isPending && data.type === 'package'" class="node-icon package-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isPending && (data as any).isGroup" class="node-icon group-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <FolderOpened />
                  </el-icon>
                  <span class="node-label" :class="{ 'group-label': (data as any).isGroup && !(data as any).isPending }" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup && !(data as any).isPending && !(data as any).mappingColor" type="info" size="small" class="group-tag" style="margin-left: 8px;">
                    组
                  </ElTag>
                  <ElTag v-if="(data as any).isPending" type="warning" size="small" effect="dark" style="margin-left: 8px;">
                    {{ (data as any).isGroup ? '待克隆' : '待克隆目录' }}
                  </ElTag>
                  
                  <!-- 操作按钮 -->
                  <el-dropdown
                    trigger="click"
                    @click.stop
                    class="node-actions"
                    @command="(command: string) => {
                      if (command === 'create-directory') {
                        handleCreateDirectory(data)
                      }
                    }"
                  >
                    <el-icon class="more-icon" @click.stop>
                      <MoreFilled />
                    </el-icon>
                    <template #dropdown>
                      <el-dropdown-menu>
                        <el-dropdown-item command="create-directory">
                          <el-icon><Plus /></el-icon>
                          创建子目录
                        </el-dropdown-item>
                      </el-dropdown-menu>
                    </template>
                  </el-dropdown>
                </div>
              </template>
            </el-tree>
            <ElEmpty v-else-if="!loadingTargetTree" description="暂无服务目录" :image-size="80" />
          </div>
        </div>

        <!-- 右侧：未选择目标应用时的提示 -->
        <div class="target-panel empty" v-else>
          <ElEmpty description="请先选择目标应用" :image-size="100" />
        </div>
      </div>

      <!-- 已添加的映射关系 -->
      <div class="mappings-section" v-if="mappings.length > 0">
        <div class="section-title">已添加的映射关系（{{ mappings.length }}）</div>
        <div class="mappings-list">
          <div
            v-for="(mapping, index) in mappings"
            :key="index"
            class="mapping-item"
          >
            <div class="mapping-content">
              <div class="mapping-source">
                <ElTag type="info">{{ mapping.sourceName || formatPath(mapping.source) }}</ElTag>
              </div>
              <div class="mapping-arrow">
                <el-icon><ArrowRight /></el-icon>
              </div>
              <div class="mapping-target">
                <ElTag type="success">{{ mapping.targetName || formatPath(mapping.target) }}</ElTag>
              </div>
            </div>
            <ElButton
              type="danger"
              :icon="Delete"
              size="small"
              circle
              @click="removeMapping(index)"
            />
          </div>
        </div>
      </div>
    </div>

    <template #footer>
      <div class="dialog-footer">
        <ElButton @click="handleCancel">取消</ElButton>
        <ElButton
          type="primary"
          :disabled="!selectedApp || mappings.length === 0"
          @click="handleSubmit"
        >
          闪电克隆（{{ mappings.length }}）
        </ElButton>
      </div>
    </template>
  </ElDialog>

  <!-- 创建服务目录对话框 -->
  <ElDialog
    v-model="createDirectoryDialogVisible"
    :title="currentParentNode ? `在「${currentParentNode.name || currentParentNode.code}」下创建服务目录` : '创建服务目录'"
    width="520px"
    :close-on-click-modal="false"
    @close="() => {
      createDirectoryForm = {
        user: selectedApp?.user || '',
        app: selectedApp?.code || '',
        name: '',
        code: '',
        parent_id: 0,
        description: '',
        tags: '',
        admins: getDefaultAdmins()  // 默认当前用户为管理员
      }
      currentParentNode = null
    }"
  >
    <ElForm :model="createDirectoryForm" label-width="90px">
      <ElFormItem label="目录名称" required>
        <ElInput
          v-model="createDirectoryForm.name"
          placeholder="请输入目录名称（如：用户管理）"
          maxlength="50"
          show-word-limit
          clearable
        />
      </ElFormItem>
      <ElFormItem label="目录代码" required>
        <ElInput
          v-model="createDirectoryForm.code"
          placeholder="请输入目录代码，如：user"
          maxlength="50"
          show-word-limit
          clearable
          @input="createDirectoryForm.code = createDirectoryForm.code.toLowerCase()"
        />
        <div class="form-tip">
          目录代码只能包含小写字母、数字和下划线
        </div>
      </ElFormItem>
      <ElFormItem label="描述">
        <ElInput
          v-model="createDirectoryForm.description"
          type="textarea"
          :rows="3"
          placeholder="请输入目录描述（可选）"
          maxlength="200"
          show-word-limit
        />
      </ElFormItem>
      <ElFormItem label="标签">
        <ElInput
          v-model="createDirectoryForm.tags"
          placeholder="请输入标签，多个标签用逗号分隔（可选）"
          maxlength="100"
          clearable
        />
      </ElFormItem>
      <ElFormItem label="管理员">
        <UsersWidget
          :field="adminsField"
          :value="adminsFieldValue"
          mode="edit"
          @update:modelValue="handleAdminsChange"
        />
        <div class="form-tip">
          <el-icon><InfoFilled /></el-icon>
          默认当前用户为管理员，可以添加其他用户
        </div>
      </ElFormItem>
    </ElForm>

    <template #footer>
      <span class="dialog-footer">
        <ElButton @click="createDirectoryDialogVisible = false">取消</ElButton>
        <ElButton type="primary" @click="handleSubmitCreateDirectory" :loading="creatingDirectory">
          创建
        </ElButton>
      </span>
    </template>
  </ElDialog>
</template>

<style scoped>
.fork-dialog :deep(.el-dialog) {
  margin: 0 auto;
  top: 50%;
  transform: translateY(-50%);
}

.fork-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.fork-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.target-app-selector {
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.fork-layout {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 20px;
  min-height: 500px;
  position: relative;
}

/* SVG 连接线层 */
.connection-lines-layer {
  position: absolute;
  top: 0;
  left: 0;
  pointer-events: none;
  overflow: visible;
}

/* 连接线流动动画 */
@keyframes flowLine {
  0% {
    stroke-dashoffset: 0;
  }
  100% {
    stroke-dashoffset: -12;
  }
}

/* Loading 图标旋转动画 */
@keyframes spin {
  0% {
    transform: rotate(0deg);
  }
  100% {
    transform: rotate(360deg);
  }
}

.connection-line {
  animation: flowLine 2s linear infinite;
}

.source-panel,
.target-panel {
  display: flex;
  flex-direction: column;
  border: 1px solid var(--el-border-color-light);
  border-radius: 4px;
  overflow: hidden;
  background: var(--el-bg-color);
}

.source-panel.empty,
.target-panel.empty {
  align-items: center;
  justify-content: center;
}

.panel-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 16px;
  background: var(--el-fill-color-lighter);
  border-bottom: 1px solid var(--el-border-color-light);
}

.panel-header h3 {
  margin: 0;
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
}

.panel-content {
  flex: 1;
  overflow: auto;
  padding: 12px;
  min-height: 400px;
}

.root-drop-zone {
  display: flex;
  align-items: center;
  gap: 8px;
  padding: 12px;
  margin-bottom: 12px;
  border: 2px dashed var(--el-border-color-light);
  border-radius: 4px;
  background: var(--el-fill-color-lighter);
  transition: all 0.2s;
  cursor: pointer;
}

.root-drop-zone:hover {
  background: var(--el-fill-color);
  border-color: var(--el-color-primary-light-7);
}

.root-drop-zone.is-drag-over {
  background: var(--el-color-primary-light-9);
  border: 2px dashed var(--el-color-primary);
  border-color: var(--el-color-primary);
}

.root-drop-zone .root-icon {
  font-size: 16px;
  width: 16px;
  height: 16px;
  color: #6366f1; /* 与目录图标颜色保持一致 */
  opacity: 0.8;
  flex-shrink: 0;
}

.root-drop-zone .root-label {
  font-size: 14px;
  color: var(--el-text-color-primary);
  font-weight: 500;
  flex: 1;
}


.source-tree {
  width: 100%;
}

.source-tree .tree-node-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  min-height: 32px;
}

.source-tree .tree-node-wrapper.is-draggable {
  cursor: move;
}

.source-tree .tree-node-wrapper.is-draggable:hover {
  background: var(--el-fill-color-lighter);
}

.source-tree .tree-node-wrapper.is-dragging {
  opacity: 0.5;
}

.arrow-panel {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 8px;
}

.arrow-icon {
  font-size: 32px;
  color: var(--el-color-primary);
}

.arrow-text {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}

.target-tree {
  width: 100%;
}

.tree-node-wrapper {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 4px 8px;
  border-radius: 4px;
  transition: all 0.2s;
  min-height: 32px;
}

.tree-node-wrapper.is-package {
  cursor: pointer;
  background-color: transparent;
}

.tree-node-wrapper.is-package:hover {
  background-color: var(--el-fill-color-light);
}

.tree-node-wrapper.is-group {
  background-color: var(--el-fill-color-lighter);
  border-left: 3px solid #909399;
  padding-left: 5px;
}

.tree-node-wrapper.is-group:hover {
  background-color: var(--el-fill-color);
}

.tree-node-wrapper.is-drag-over {
  background: var(--el-color-primary-light-9);
  border: 2px dashed var(--el-color-primary);
}

.tree-node-wrapper.is-pending {
  font-weight: 500;
}

.tree-node-wrapper.is-pending .node-icon {
  opacity: 0.9;
}

.tree-node-wrapper.is-pending .node-label {
  font-weight: 500;
}

.tree-node-wrapper.has-mapping {
  font-weight: 500;
  border-radius: 4px;
  transition: all 0.2s;
}

.tree-node-wrapper.has-mapping .node-icon {
  opacity: 0.9;
}

.tree-node-wrapper.has-mapping .node-label {
  font-weight: 500;
}

.node-icon {
  font-size: 16px;
  width: 16px;
  height: 16px;
  flex-shrink: 0;
  transition: color 0.2s ease;
}

.node-icon.package-icon {
  color: #6366f1; /* indigo-500，与根服务目录保持一致 */
  opacity: 0.8;
}

.node-icon.group-icon {
  color: #909399; /* 灰色，用于区分函数组 */
  opacity: 0.9;
}

.node-icon.fx-icon {
  font-size: 12px;
  font-weight: 600;
  font-family: 'SF Mono', 'Monaco', 'Inconsolata', 'Roboto Mono', monospace;
  font-style: italic;
  color: #6366f1;
  opacity: 0.8;
}

.node-label {
  font-size: 14px;
  color: var(--el-text-color-primary);
  flex: 1;
}

.node-label.group-label {
  font-weight: 500;
  color: var(--el-text-color-regular);
}

.group-tag {
  font-size: 11px;
  margin-left: 8px;
}

.node-actions {
  flex-shrink: 0;
  opacity: 0;
  transition: opacity 0.2s;
  margin-left: 8px;
}

.tree-node-wrapper:hover .node-actions {
  opacity: 1;
}

.more-icon {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  padding: 4px;
  transition: all 0.2s;
}

.more-icon:hover {
  color: var(--el-color-primary);
  background-color: var(--el-fill-color);
  border-radius: 2px;
}

.form-tip {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 4px;
}

.mappings-section {
  margin-top: 20px;
  padding-top: 20px;
  border-top: 1px solid var(--el-border-color-light);
}

.section-title {
  font-size: 14px;
  font-weight: 600;
  margin-bottom: 12px;
  color: var(--el-text-color-primary);
}

.mappings-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.mapping-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
  transition: all 0.2s;
}

.mapping-item:hover {
  background: var(--el-fill-color);
}

.mapping-content {
  display: flex;
  align-items: center;
  gap: 12px;
  flex: 1;
}

.mapping-source,
.mapping-target {
  flex: 1;
}

.mapping-arrow {
  color: var(--el-color-primary);
  font-size: 18px;
}

.dialog-footer {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
}

/* 🔥 克隆通知样式优化，确保文字清晰可见 */
:deep(.fork-progress-notification),
:deep(.fork-success-notification) {
  background-color: var(--el-bg-color) !important;
  border: 1px solid var(--el-border-color-light) !important;
  box-shadow: 0 2px 12px 0 rgba(0, 0, 0, 0.1) !important;
}

:deep(.fork-progress-notification .el-notification__title),
:deep(.fork-success-notification .el-notification__title) {
  color: #303133 !important;
  font-weight: 600 !important;
  font-size: 16px !important;
}

:deep(.fork-progress-notification .el-notification__content),
:deep(.fork-success-notification .el-notification__content) {
  color: #303133 !important;
}

/* 🔥 强制设置通知内容中的所有文字为深色，确保清晰可见 */
:deep(.fork-progress-notification .fork-notification-content),
:deep(.fork-success-notification .fork-notification-content) {
  color: #303133 !important;
}

:deep(.fork-progress-notification .fork-notification-text),
:deep(.fork-success-notification .fork-notification-text) {
  color: #303133 !important;
}

:deep(.fork-progress-notification .el-notification__content p),
:deep(.fork-success-notification .el-notification__content p) {
  color: #303133 !important;
  margin: 0 !important;
}

:deep(.fork-progress-notification .el-notification__content span),
:deep(.fork-success-notification .el-notification__content span) {
  color: #303133 !important;
}

/* 🔥 确保通知内容中的所有文字都是深色，清晰可见 - 使用更强制的方式 */
:deep(.fork-progress-notification .el-notification__content *),
:deep(.fork-success-notification .el-notification__content *) {
  color: #303133 !important;
}

:deep(.fork-progress-notification .fork-notification-content *),
:deep(.fork-success-notification .fork-notification-content *) {
  color: #303133 !important;
}
</style>
