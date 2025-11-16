<script setup lang="ts">
import { ref, computed, watch, h } from 'vue'
import { useRouter } from 'vue-router'
import { ElDialog, ElSelect, ElOption, ElInput, ElButton, ElMessage, ElNotification, ElTag, ElEmpty, ElTree, ElForm, ElFormItem, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { Search, Delete, ArrowRight, Folder, FolderOpened, Plus, MoreFilled, Document } from '@element-plus/icons-vue'
import { getAppList } from '@/api/app'
import { getServiceTree, createServiceTree } from '@/api/service-tree'
import { forkFunctionGroup } from '@/api/function'
import { generateGroupId, createGroupNode, groupFunctionsByCode, getGroupName, type ExtendedServiceTree } from '@/utils/tree-utils'
import type { App, ServiceTree as ServiceTreeType, CreateServiceTreeRequest } from '@/types'

interface Props {
  modelValue: boolean
  sourceFullGroupCode?: string  // 源函数组的 full_group_code，格式：/user/app/package/group_code（可选）
  sourceGroupName?: string      // 源函数组名称（用于显示）
  sourceApp?: App               // 源应用（可选，如果提供则不需要解析）
  currentApp?: App              // 当前应用（当没有 sourceFullGroupCode 时，使用当前应用作为源应用）
}

interface Emits {
  (e: 'update:modelValue', value: boolean): void
  (e: 'success'): void  // Fork 成功后的回调
}

const props = defineProps<Props>()
const emit = defineEmits<Emits>()

// 对话框显示状态
const dialogVisible = computed({
  get: () => props.modelValue,
  set: (value) => emit('update:modelValue', value)
})

// 从 full_group_code 解析源应用信息
const parseSourceApp = (fullGroupCode?: string): { user: string; app: string } | null => {
  if (!fullGroupCode) {
    return null
  }
  // 格式：/user/app/package/group_code
  const parts = fullGroupCode.split('/').filter(Boolean)
  if (parts.length >= 2) {
    return {
      user: parts[0],
      app: parts[1]
    }
  }
  return null
}

// 源应用信息
const sourceAppInfo = computed(() => {
  if (props.sourceApp) {
    return {
      user: props.sourceApp.user,
      app: props.sourceApp.code,
      appObj: props.sourceApp
    }
  }
  if (props.sourceFullGroupCode) {
    const parsed = parseSourceApp(props.sourceFullGroupCode)
    if (parsed) {
      return {
        ...parsed,
        appObj: null
      }
    }
  }
  // 如果没有 sourceFullGroupCode，使用当前应用作为源应用
  if (props.currentApp) {
    return {
      user: props.currentApp.user,
      app: props.currentApp.code,
      appObj: props.currentApp
    }
  }
  return null
})

// 应用搜索关键词
const appSearchKeyword = ref('')
// 应用列表
const appList = ref<App[]>([])
const loadingApps = ref(false)
// 选中的目标应用
const selectedApp = ref<App | null>(null)

// 源应用的服务目录树（显示所有函数组）
const sourceServiceTree = ref<ServiceTreeType[]>([])
const loadingSourceTree = ref(false)

// 目标应用的服务目录树（只显示 package 类型）
const targetServiceTree = ref<ServiceTreeType[]>([])
const loadingTargetTree = ref(false)

// 映射关系列表
interface ForkMapping {
  source: string  // 源函数组的 full_group_code
  target: string   // 目标目录的 full_code_path
  targetName?: string  // 目标目录名称（用于显示）
  sourceName?: string  // 源函数组名称（用于显示）
}
const mappings = ref<ForkMapping[]>([])

// 拖拽相关状态
const draggedNode = ref<ServiceTreeType | null>(null)
const dragOverNode = ref<ServiceTreeType | null>(null) // null 表示根目录
const isDragging = ref(false)

// 加载应用列表
const loadAppList = async (keyword?: string) => {
  try {
    loadingApps.value = true
    const apps = await getAppList(200, keyword)
    appList.value = apps
  } catch (error) {
    console.error('加载应用列表失败:', error)
    ElMessage.error('加载应用列表失败')
    appList.value = []
  } finally {
    loadingApps.value = false
  }
}

// 搜索应用（防抖）
let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(appSearchKeyword, (keyword) => {
  if (searchTimer) {
    clearTimeout(searchTimer)
  }
  searchTimer = setTimeout(() => {
    loadAppList(keyword || undefined)
  }, 300)
})

// 加载源应用的服务目录树
const loadSourceServiceTree = async () => {
  if (!sourceAppInfo.value) {
    return
  }
  
  try {
    loadingSourceTree.value = true
    // 加载所有类型的节点，然后在前端过滤出函数组
    const tree = await getServiceTree(sourceAppInfo.value.user, sourceAppInfo.value.app)
    sourceServiceTree.value = tree || []
  } catch (error) {
    console.error('加载源服务目录树失败:', error)
    ElMessage.error('加载源服务目录树失败')
    sourceServiceTree.value = []
  } finally {
    loadingSourceTree.value = false
  }
}

// 选择目标应用
const handleSelectApp = async (app: App | null) => {
  selectedApp.value = app
  
  if (app) {
    // 加载目标应用的服务目录树（只显示 package 类型）
    await loadTargetServiceTree(app)
  } else {
    targetServiceTree.value = []
  }
}

// 加载目标应用的服务目录树（加载所有类型，以便显示函数组）
const loadTargetServiceTree = async (app: App) => {
  try {
    loadingTargetTree.value = true
    // 加载所有类型，然后在前端处理成分组显示
    const tree = await getServiceTree(app.user, app.code)
    targetServiceTree.value = tree || []
  } catch (error) {
    console.error('加载目标服务目录树失败:', error)
    ElMessage.error('加载目标服务目录树失败')
    targetServiceTree.value = []
  } finally {
    loadingTargetTree.value = false
  }
}

// 递归收集目录下的所有函数组（有 full_group_code 的节点）
const collectFunctionGroups = (node: ServiceTreeType): ServiceTreeType[] => {
  const groups: ServiceTreeType[] = []
  
  // 如果是函数组节点（isGroup），添加到结果中
  if ((node as any).isGroup && node.full_group_code) {
    groups.push(node)
  }
  
  // 递归处理子节点
  if (node.children && node.children.length > 0) {
    node.children.forEach(child => {
      groups.push(...collectFunctionGroups(child))
    })
  }
  
  return groups
}

// 递归创建目录结构（包括所有子目录）
const createDirectoryRecursively = async (
  sourceNode: ServiceTreeType,
  targetParentId: number,
  targetParentPath: string
): Promise<ServiceTreeType> => {
  // 1. 创建当前目录
  const newDirectory = await createServiceTree({
    user: selectedApp.value!.user,
    app: selectedApp.value!.code,
    name: sourceNode.name,
    code: sourceNode.code,
    parent_id: targetParentId,
    description: sourceNode.description || '',
    tags: sourceNode.tags || ''
  })
  
  // 2. 递归创建子目录（只处理 package 类型的子节点）
  if (sourceNode.children && sourceNode.children.length > 0) {
    const packageChildren = sourceNode.children.filter(
      child => child.type === 'package' && !(child as any).isGroup
    )
    
    for (const childPackage of packageChildren) {
      await createDirectoryRecursively(
        childPackage,
        newDirectory.id,
        newDirectory.full_code_path
      )
    }
  }
  
  return newDirectory
}

// 拖拽开始 - 源节点
const handleDragStart = (node: ServiceTreeType, event: DragEvent) => {
  // 允许拖拽：
  // 1. 函数组节点（isGroup 且有 full_group_code）
  // 2. 服务目录节点（type === 'package' 且不是函数组）
  const isGroup = (node as any).isGroup && node.full_group_code && node.full_group_code.trim() !== ''
  const isPackage = node.type === 'package' && !(node as any).isGroup
  
  if (!isGroup && !isPackage) {
    event.preventDefault()
    return false
  }
  
  draggedNode.value = node
  isDragging.value = true
  
  // 设置拖拽数据
  if (event.dataTransfer) {
    event.dataTransfer.effectAllowed = 'move'
    const data = isGroup ? node.full_group_code : `__package__${node.id}`
    event.dataTransfer.setData('text/plain', data)
  }
  
  // 添加拖拽样式
  if (event.target) {
    (event.target as HTMLElement).style.opacity = '0.5'
  }
}

// 拖拽结束 - 源节点
const handleDragEnd = (event: DragEvent) => {
  isDragging.value = false
  draggedNode.value = null
  dragOverNode.value = null
  
  // 恢复样式
  if (event.target) {
    (event.target as HTMLElement).style.opacity = '1'
  }
}

// 拖拽悬停 - 目标节点（可以是 package 节点或根目录）
const handleDragOver = (node: ServiceTreeType | null, event: DragEvent) => {
  // 如果是根目录（node 为 null），允许拖拽
  if (node === null) {
    event.preventDefault()
    dragOverNode.value = null // 使用 null 表示根目录
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'move'
    }
    return true
  }
  
  // 只允许拖拽到 package 类型的节点（且不是函数组）
  if (node.type !== 'package' || (node as any).isGroup) {
    if (event.dataTransfer) {
      event.dataTransfer.dropEffect = 'none'
    }
    return false
  }
  
  event.preventDefault()
  dragOverNode.value = node
  
  if (event.dataTransfer) {
    event.dataTransfer.dropEffect = 'move'
  }
  
  return true
}

// 拖拽离开 - 目标节点
const handleDragLeave = () => {
  dragOverNode.value = null
}

// 拖拽放置 - 目标节点（可以是 package 节点或根目录，只添加映射关系，不立即执行 Fork）
const handleDrop = async (node: ServiceTreeType | null, event: DragEvent) => {
  event.preventDefault()
  
  // 保存拖拽的节点信息
  const sourceNode = draggedNode.value
  if (!sourceNode || !selectedApp.value) {
    dragOverNode.value = null
    return
  }
  
  // node 为 null 表示根目录，允许拖拽
  // 否则只允许拖拽到 package 类型的节点（且不是函数组）
  if (node !== null && (node.type !== 'package' || (node as any).isGroup)) {
    ElMessage.warning('只能拖拽到 package 类型的目录或根目录')
    dragOverNode.value = null
    return
  }
  
  const isSourceGroup = (sourceNode as any).isGroup && sourceNode.full_group_code
  const isSourcePackage = sourceNode.type === 'package' && !(sourceNode as any).isGroup
  
  // 情况1：拖拽函数组
  if (isSourceGroup) {
    // 检查源节点是否有 full_group_code
    if (!sourceNode.full_group_code) {
      ElMessage.warning('源函数组没有 full_group_code，无法克隆')
      dragOverNode.value = null
      draggedNode.value = null
      return
    }
    
    // 计算目标路径：如果是根目录（node 为 null），使用应用根路径；否则使用节点的 full_code_path
    // 注意：targetPath 应该是 package 的路径，不包含 group_code
    const targetPath = node === null 
      ? `/${selectedApp.value.user}/${selectedApp.value.code}` 
      : node.full_code_path
    const targetName = node === null ? '根目录' : node.name
    
    // 检查是否已经存在相同的映射
    const existingMapping = mappings.value.find(
      m => m.source === sourceNode.full_group_code && m.target === targetPath
    )
    
    if (existingMapping) {
      ElMessage.warning('该映射关系已存在')
      dragOverNode.value = null
      draggedNode.value = null
      return
    }
    
    // 只添加映射关系，不立即执行 Fork
    mappings.value.push({
      source: sourceNode.full_group_code,
      target: targetPath,
      targetName: targetName,
      sourceName: sourceNode.group_name || sourceNode.name || ''
    })
    
    ElMessage.success('已添加映射关系，点击"确认克隆"提交')
  }
  // 情况2：拖拽服务目录
  else if (isSourcePackage) {
    try {
      // 1. 在目标目录下递归创建目录结构（包括所有子目录）
      // 如果是根目录（node 为 null），parent_id 为 0；否则使用 node.id
      const parentId = node === null ? 0 : Number(node.id)
      const parentPath = node === null 
        ? `${selectedApp.value.user}/${selectedApp.value.code}`
        : node.full_code_path
      
      console.log(`[FunctionForkDialog] 开始递归创建目录结构: ${sourceNode.name}`)
      const newDirectory = await createDirectoryRecursively(sourceNode, parentId, parentPath)
      
      ElMessage.success(`已创建目录「${sourceNode.name}」及其子目录`)
      
      // 2. 收集源目录下的所有函数组（递归收集，包括所有子目录中的函数组）
      const functionGroups = collectFunctionGroups(sourceNode)
      
      if (functionGroups.length === 0) {
        ElMessage.warning('该目录下没有函数组')
        dragOverNode.value = null
        draggedNode.value = null
        return
      }
      
      // 3. 为每个函数组添加映射关系
      // 需要找到每个函数组对应的目标目录路径
      const newMappings: typeof mappings.value = []
      
      // 创建一个辅助函数，根据源路径找到对应的目标路径（只到 package 目录）
      const findTargetPath = (sourceGroupCode: string): string => {
        // sourceGroupCode 格式: /user/app/path/to/package/group_code
        // 需要找到对应的目标路径（只到 package 目录，不包括 group_code）
        // 例如：源路径 /luobei/testfork666/tools/tools/tools_cashier
        //      目标路径应该是 /luobei/testfork777/tools/tools（去掉最后的 group_code）
        
        // 提取源路径中相对于源目录的部分
        const sourcePath = sourceNode.full_code_path
        if (sourceGroupCode.startsWith(sourcePath)) {
          // 获取相对路径部分（包括 package 和 group_code）
          const relativePath = sourceGroupCode.substring(sourcePath.length)
          // 去掉最后的 group_code 部分（最后一个路径段）
          // 例如：/tools/tools_cashier -> /tools
          const relativePathSegments = relativePath.split('/').filter(Boolean)
          if (relativePathSegments.length > 0) {
            // 去掉最后一个路径段（group_code），只保留 package 路径
            relativePathSegments.pop()
            const packageRelativePath = relativePathSegments.length > 0 
              ? '/' + relativePathSegments.join('/')
              : ''
            // 构建目标路径（只到 package 目录）
            return newDirectory.full_code_path + packageRelativePath
          }
          // 如果没有相对路径，直接使用新创建的目录
          return newDirectory.full_code_path
        }
        // 如果无法匹配，尝试从 sourceGroupCode 中提取 package 路径
        // sourceGroupCode 格式: /user/app/path/to/package/group_code
        // 需要去掉最后的 group_code
        const segments = sourceGroupCode.split('/').filter(Boolean)
        if (segments.length > 0) {
          // 去掉最后一个路径段（group_code）
          segments.pop()
          // 构建目标路径：/user/app/.../package（去掉最后的 group_code）
          // 但需要替换 user/app 为目标应用的 user/app
          const targetUser = selectedApp.value.user
          const targetApp = selectedApp.value.code
          // 保留从第3个路径段开始到倒数第二个路径段（去掉 user, app, group_code）
          if (segments.length >= 2) {
            const packageSegments = segments.slice(2) // 去掉 user 和 app
            return `/${targetUser}/${targetApp}${packageSegments.length > 0 ? '/' + packageSegments.join('/') : ''}`
          }
        }
        // 如果无法匹配，使用新创建的目录作为目标
        return newDirectory.full_code_path
      }
      
      functionGroups.forEach(group => {
        if (group.full_group_code) {
          // 找到对应的目标路径
          const targetPath = findTargetPath(group.full_group_code)
          
          // 检查是否已经存在相同的映射
          const existingMapping = mappings.value.find(
            m => m.source === group.full_group_code && m.target === targetPath
          )
          
          if (!existingMapping) {
            // 从 targetPath 中提取目录名称（最后一个路径段）
            const targetPathSegments = targetPath.split('/').filter(Boolean)
            const targetDirName = targetPathSegments.length > 2 
              ? targetPathSegments[targetPathSegments.length - 2] 
              : newDirectory.name
            
            newMappings.push({
              source: group.full_group_code,
              target: targetPath,
              targetName: targetDirName,
              sourceName: group.group_name || group.name || ''
            })
          }
        }
      })
      
      if (newMappings.length > 0) {
        mappings.value.push(...newMappings)
        ElMessage.success(`已添加 ${newMappings.length} 个函数组的映射关系，点击"确认克隆"提交`)
      } else {
        ElMessage.warning('所有函数组的映射关系已存在')
      }
      
      // 4. 刷新目标服务目录树，以便看到新创建的目录
      await loadTargetServiceTree(selectedApp.value)
    } catch (error: any) {
      console.error('创建目录失败:', error)
      ElMessage.error(error?.message || '创建目录失败')
    }
  } else {
    ElMessage.warning('只能拖拽函数组或服务目录')
    dragOverNode.value = null
    draggedNode.value = null
    return
  }
  
  // 清空拖拽状态
  dragOverNode.value = null
  draggedNode.value = null
}

// 删除映射关系
const removeMapping = (index: number) => {
  mappings.value.splice(index, 1)
}

// 创建服务目录相关
const createDirectoryDialogVisible = ref(false)
const createDirectoryForm = ref<CreateServiceTreeRequest>({
  user: '',
  app: '',
  name: '',
  code: '',
  parent_id: 0,
  description: '',
  tags: ''
})
const creatingDirectory = ref(false)
const currentParentNode = ref<ServiceTreeType | null>(null)

// 打开创建目录对话框
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
    tags: ''
  }
  createDirectoryDialogVisible.value = true
}

// 提交创建目录
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
    
    // 关闭对话框
    createDirectoryDialogVisible.value = false
    currentParentNode.value = null
    
    // 刷新目标应用的服务目录树
    await loadTargetServiceTree(selectedApp.value)
  } catch (error: any) {
    console.error('创建目录失败:', error)
    ElMessage.error(error?.message || '创建目录失败')
  } finally {
    creatingDirectory.value = false
  }
}

const router = useRouter()

// 提交 Fork（批量提交所有映射关系，后台克隆）
const handleSubmit = async () => {
  // 验证
  if (!selectedApp.value) {
    ElMessage.warning('请选择目标应用')
    return
  }
  
  if (mappings.value.length === 0) {
    ElMessage.warning('请至少添加一个映射关系')
    return
  }
  
  // 先关闭对话框并显示加载提示
  dialogVisible.value = false
  ElMessage.info('正在提交克隆任务...')
  
  try {
    // 构建 source_to_target_map
    const sourceToTargetMap: Record<string, string> = {}
    mappings.value.forEach(mapping => {
      sourceToTargetMap[mapping.source] = mapping.target
    })
    
    // 先保存目标应用信息和映射关系，避免异步操作后丢失
    const targetApp = { ...selectedApp.value }
    const savedMappings = [...mappings.value] // 保存映射关系的副本
    
    // 调用 Fork API（批量提交，后台克隆）
    await forkFunctionGroup({
      source_to_target_map: sourceToTargetMap,
      target_app_id: selectedApp.value.id
    })
    
    // 使用 ElNotification 提示成功，并支持跳转
    if (targetApp && targetApp.user && targetApp.code) {
      ElNotification({
        title: '克隆成功',
        message: h('div', { style: 'line-height: 1.6;' }, [
          h('p', { style: 'margin: 0 0 8px 0; color: #303133;' }, `成功提交 ${savedMappings.length} 个函数组的克隆任务`),
          h('p', { style: 'margin: 0 0 12px 0; color: #909399; font-size: 12px;' }, '克隆操作正在后台执行，完成后即可使用'),
          h(ElButton, {
            type: 'primary',
            size: 'small',
            onClick: () => {
              // 在新窗口打开目标应用，并传递新克隆的路径信息
              const forkedPaths = savedMappings.map((m: ForkMapping) => m.target).join(',')
              console.log('[FunctionForkDialog] 准备跳转，路径列表:', savedMappings.map((m: ForkMapping) => m.target))
              console.log('[FunctionForkDialog] 编码后的路径:', forkedPaths)
              const url = `/workspace/${targetApp.user}/${targetApp.code}${forkedPaths ? `?forked=${encodeURIComponent(forkedPaths)}` : ''}`
              console.log('[FunctionForkDialog] 完整 URL:', url)
              // 在新窗口打开
              window.open(url, '_blank')
            }
          }, () => `跳转到 ${targetApp.name || targetApp.code}`)
        ]),
        type: 'success',
        duration: 5000,
        position: 'top-right'
      })
    }
    
    // 触发成功回调
    emit('success')
  } catch (error: any) {
    console.error('Fork 失败:', error)
    ElMessage.error(error?.message || '克隆操作失败')
    // 如果失败，重新打开对话框
    dialogVisible.value = true
  }
}

// 取消
const handleCancel = () => {
  dialogVisible.value = false
}

// 重置表单
const resetForm = () => {
  appSearchKeyword.value = ''
  selectedApp.value = null
  sourceServiceTree.value = []
  targetServiceTree.value = []
  mappings.value = []
  draggedNode.value = null
  dragOverNode.value = null
  isDragging.value = false
  createDirectoryDialogVisible.value = false
  createDirectoryForm.value = {
    user: '',
    app: '',
    name: '',
    code: '',
    parent_id: 0,
    description: '',
    tags: ''
  }
  currentParentNode.value = null
}

// 监听对话框显示状态
watch(dialogVisible, (visible) => {
  if (visible) {
    resetForm()
    loadAppList()
    loadSourceServiceTree()
  }
})

// 格式化显示路径（去掉用户名和应用名）
const formatPath = (path: string) => {
  const parts = path.split('/').filter(Boolean)
  if (parts.length >= 2) {
    return parts.slice(2).join('/')
  }
  return path
}

// 将源服务目录树处理为带函数组的结构（参考 ServiceTreePanel 的逻辑）
const groupedSourceTree = computed(() => {
  const processNode = (node: ServiceTreeType): ServiceTreeType => {
    // 如果是 package 且有子节点，需要分组处理
    if (node.type === 'package' && node.children && node.children.length > 0) {
      // 分离函数和包
      const functions = node.children.filter(child => child.type === 'function')
      const packages = node.children.filter(child => child.type === 'package')
      
      // 按 full_group_code 分组函数
      const { grouped: groupedFunctions, ungrouped: ungroupedFunctions } = groupFunctionsByCode(functions)
      
      // 构建新的 children 数组
      const newChildren: ServiceTreeType[] = []
      
      // 1. 先添加包（保持原有顺序）
      packages.forEach(pkg => {
        newChildren.push(processNode(pkg))
      })
      
      // 2. 添加分组后的函数（创建虚拟函数组节点）
      groupedFunctions.forEach((funcs, groupCode) => {
        const groupName = getGroupName(funcs, groupCode)
        const groupNode = createGroupNode(groupCode, groupName, node, true)
        // 函数组下包含函数节点
        groupNode.children = funcs.map(func => processNode(func))
        newChildren.push(groupNode)
      })
      
      // 3. 添加未分组的函数
      ungroupedFunctions.forEach(func => {
        newChildren.push(processNode(func))
      })
      
      return {
        ...node,
        children: newChildren
      }
    }
    
    // 如果是函数或没有子节点，直接返回
    return node
  }
  
  return sourceServiceTree.value.map(node => processNode(node))
})

// 将目标服务目录树处理为带函数组的结构（只显示 package 和函数组，不显示单个函数）
const groupedTargetTree = computed(() => {
  // 创建一个映射，用于快速查找某个目录下已添加映射的函数组
  const mappingsByTarget = new Map<string, ForkMapping[]>()
  mappings.value.forEach(mapping => {
    if (!mappingsByTarget.has(mapping.target)) {
      mappingsByTarget.set(mapping.target, [])
    }
    mappingsByTarget.get(mapping.target)!.push(mapping)
  })
  
  
  const processNode = (node: ServiceTreeType): ServiceTreeType | null => {
    // 如果是函数节点，直接返回 null（不显示）
    if (node.type === 'function') {
      return null
    }
    
    // 如果是 package 且有子节点，需要分组处理
    if (node.type === 'package' && node.children && node.children.length > 0) {
      // 分离函数和包
      const functions = node.children.filter(child => child.type === 'function')
      const packages = node.children.filter(child => child.type === 'package')
      
      // 按 full_group_code 分组函数（只处理有 full_group_code 的函数）
      const { grouped: groupedFunctions } = groupFunctionsByCode(functions)
      // 注意：未分组的函数不显示（右侧只显示 package 和函数组）
      
      // 构建新的 children 数组
      const newChildren: ServiceTreeType[] = []
      
      // 1. 先添加包（保持原有顺序）
      packages.forEach(pkg => {
        const processed = processNode(pkg)
        if (processed) {
          newChildren.push(processed)
        }
      })
      
      // 2. 添加分组后的函数（创建虚拟函数组节点）
      groupedFunctions.forEach((funcs, groupCode) => {
        const groupName = getGroupName(funcs, groupCode)
        const groupNode = createGroupNode(groupCode, groupName, node, false)
        // 函数组下不显示函数节点（右侧只显示函数组，不显示函数）
        newChildren.push(groupNode)
      })
      
      // 3. 添加已添加映射但尚未 fork 的函数组（虚拟节点）
      const pendingMappings = mappingsByTarget.get(node.full_code_path) || []
      pendingMappings.forEach(mapping => {
        // 检查是否已经存在对应的函数组节点（已 fork 的）
        const existingGroup = newChildren.find(
          child => (child as any).isGroup && child.full_group_code === mapping.source
        )
        
        // 如果不存在，说明是已添加映射但尚未 fork 的函数组，需要显示
        if (!existingGroup) {
          const groupCode = mapping.source
          const groupName = mapping.sourceName || getGroupName([], groupCode)
          
          // 创建虚拟函数组节点（待 fork）
          const pendingGroupNode = createGroupNode(groupCode, groupName, node, false)
          pendingGroupNode.isPending = true
          
          newChildren.push(pendingGroupNode)
        }
      })
      
      return {
        ...node,
        children: newChildren
      }
    }
    
    // 如果是 package 但没有子节点，也需要检查是否有待克隆的函数组
    if (node.type === 'package') {
      const pendingMappings = mappingsByTarget.get(node.full_code_path) || []
      if (pendingMappings.length > 0) {
        // 有待克隆的函数组，需要创建 children 来显示它们
        const newChildren: ServiceTreeType[] = []
        
        pendingMappings.forEach(mapping => {
          const groupCode = mapping.source
          const groupName = mapping.sourceName || groupCode.split('/').pop() || groupCode
          
          // 生成唯一的负数 ID
          let hash = 0
          for (let i = 0; i < groupCode.length; i++) {
            const char = groupCode.charCodeAt(i)
            hash = ((hash << 5) - hash) + char
            hash = hash & hash
          }
          const groupId = -Math.abs(hash || Date.now())
          
          // 创建虚拟函数组节点（待 fork）
          const pendingGroupNode: ServiceTreeType = {
            id: groupId,
            name: groupName,
            code: `__group__${groupCode}`,
            parent_id: node.id,
            type: 'package',
            description: '',
            tags: '',
            app_id: node.app_id,
            ref_id: 0,
            full_code_path: `${node.full_code_path}/__group__${groupCode}`,
            full_group_code: groupCode,
            group_name: groupName,
            created_at: '',
            updated_at: '',
            children: [],
            // 标记为分组节点和待 fork 节点
            isGroup: true,
            isPending: true
          } as ServiceTreeType & { isGroup?: boolean; isPending?: boolean }
          
          console.log(`[groupedTargetTree] 为无子节点的目录 ${node.name} 添加待克隆函数组: ${groupName} (${groupCode})`)
          newChildren.push(pendingGroupNode)
        })
        
        return {
          ...node,
          children: newChildren
        }
      }
    }
    
    // 如果是 package 但没有子节点且没有待克隆函数组，直接返回
    return node
  }
  
  const processedTree = targetServiceTree.value.map(node => processNode(node)).filter(node => node !== null) as ServiceTreeType[]
  
  // 4. 处理根目录下的待克隆函数组（如果目标路径是应用根路径）
  const rootPath = selectedApp.value ? `${selectedApp.value.user}/${selectedApp.value.code}` : ''
  if (rootPath) {
    const rootMappings = mappingsByTarget.get(rootPath) || []
    const rootPendingGroups: ServiceTreeType[] = []
    
    rootMappings.forEach(mapping => {
      // 检查是否已经存在对应的函数组节点（在根目录下）
      const existingGroup = processedTree.find(
        node => (node as any).isGroup && node.full_group_code === mapping.source
      )
      
      // 如果不存在，说明是已添加映射但尚未 fork 的函数组，需要显示在根目录
      if (!existingGroup) {
        const groupCode = mapping.source
        const groupName = mapping.sourceName || groupCode.split('/').pop() || groupCode
        
        // 生成唯一的负数 ID
        let hash = 0
        for (let i = 0; i < groupCode.length; i++) {
          const char = groupCode.charCodeAt(i)
          hash = ((hash << 5) - hash) + char
          hash = hash & hash
        }
        const groupId = -Math.abs(hash || Date.now())
        
        // 创建虚拟函数组节点（待 fork，在根目录）
        const pendingGroupNode: ServiceTreeType = {
          id: groupId,
          name: groupName,
          code: `__group__${groupCode}`,
          parent_id: 0,
          type: 'package',
          description: '',
          tags: '',
          app_id: selectedApp.value?.id || 0,
          ref_id: 0,
          full_code_path: `${rootPath}/__group__${groupCode}`,
          full_group_code: groupCode,
          group_name: groupName,
          created_at: '',
          updated_at: '',
          children: [],
          // 标记为分组节点和待 fork 节点
          isGroup: true,
          isPending: true
        } as ServiceTreeType & { isGroup?: boolean; isPending?: boolean }
        
        rootPendingGroups.push(pendingGroupNode)
      }
    })
    
    // 将根目录下的待克隆函数组添加到树的最前面
    if (rootPendingGroups.length > 0) {
      processedTree.unshift(...rootPendingGroups)
    }
  }
  
  return processedTree
})
</script>

<template>
  <ElDialog
    v-model="dialogVisible"
    title="克隆函数组"
    width="1200px"
    :close-on-click-modal="false"
    @close="resetForm"
    class="fork-dialog"
  >
    <div class="fork-dialog-content">
      <!-- 顶部：选择目标应用 -->
      <div class="target-app-selector">
        <div class="selector-label">选择目标应用：</div>
        <ElInput
          v-model="appSearchKeyword"
          placeholder="搜索应用名称或代码"
          clearable
          style="width: 200px; margin-right: 12px;"
          :prefix-icon="Search"
        />
        <ElSelect
          v-model="selectedApp"
          placeholder="请选择目标应用"
          filterable
          :loading="loadingApps"
          style="width: 300px"
          @change="handleSelectApp"
        >
          <ElOption
            v-for="app in appList"
            :key="app.id"
            :label="`${app.name} (${app.code})`"
            :value="app"
          />
        </ElSelect>
      </div>

      <!-- 左右分栏布局 -->
      <div class="fork-layout" v-if="sourceAppInfo">
        <!-- 左侧：源应用的服务目录树（树形结构） -->
        <div class="source-panel">
          <div class="panel-header">
            <h3>源应用：{{ sourceAppInfo.user }}/{{ sourceAppInfo.app }}</h3>
            <ElTag type="info" size="small">拖拽函数组到右侧</ElTag>
          </div>
          <div class="panel-content" v-loading="loadingSourceTree">
            <el-tree
              v-if="groupedSourceTree.length > 0"
              :data="groupedSourceTree"
              :props="{ children: 'children', label: 'name' }"
              :expand-on-click-node="false"
              default-expand-all
              class="source-tree"
            >
              <template #default="{ node, data }">
                <div
                  class="tree-node-wrapper"
                  :class="{
                    'is-draggable': ((data as any).isGroup && data.full_group_code && data.full_group_code.trim() !== '') || (data.type === 'package' && !(data as any).isGroup),
                    'is-dragging': draggedNode?.id === data.id,
                    'is-source': data.full_group_code === sourceFullGroupCode,
                    'is-group': (data as any).isGroup,
                    'is-package': data.type === 'package' && !(data as any).isGroup
                  }"
                  :draggable="((data as any).isGroup && data.full_group_code && data.full_group_code.trim() !== '') || (data.type === 'package' && !(data as any).isGroup)"
                  @dragstart="((data as any).isGroup && data.full_group_code) || (data.type === 'package' && !(data as any).isGroup) ? handleDragStart(data, $event) : null"
                  @dragend="handleDragEnd"
                >
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup" class="node-icon">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup" class="node-icon">
                    <FolderOpened />
                  </el-icon>
                  <span v-else class="node-icon fx-icon">fx</span>
                  <span class="node-label">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup" type="info" size="small" style="margin-left: 8px;">
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

        <!-- 右侧：目标应用的服务目录树（只显示 package） -->
        <div class="target-panel" v-if="selectedApp">
          <div class="panel-header">
            <h3>目标应用：{{ selectedApp.user }}/{{ selectedApp.code }}</h3>
            <ElTag type="success" size="small">拖拽到这里</ElTag>
          </div>
          <div class="panel-content" v-loading="loadingTargetTree">
            <!-- 根目录拖拽区域 -->
            <div
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
            >
              <template #default="{ node, data }">
                <div
                  class="tree-node-wrapper"
                  :class="{
                    'is-drag-over': dragOverNode?.id === data.id,
                    'is-package': data.type === 'package' && !(data as any).isGroup,
                    'is-group': (data as any).isGroup
                  }"
                  @dragover="data.type === 'package' && !(data as any).isGroup ? handleDragOver(data, $event) : null"
                  @dragleave="handleDragLeave"
                  @drop="data.type === 'package' && !(data as any).isGroup ? handleDrop(data, $event) : null"
                >
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup" class="node-icon">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup" class="node-icon">
                    <FolderOpened />
                  </el-icon>
                  <span class="node-label">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup" type="info" size="small" style="margin-left: 8px;">
                    组
                  </ElTag>
                  <ElTag v-if="(data as any).isPending" type="warning" size="small" style="margin-left: 8px;">
                    待克隆
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
          确认克隆（{{ mappings.length }}）
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
        tags: ''
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
.fork-dialog :deep(.el-dialog__body) {
  padding: 20px;
}

.fork-dialog-content {
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.target-app-selector {
  display: flex;
  align-items: center;
  padding: 16px;
  background: var(--el-fill-color-lighter);
  border-radius: 4px;
}

.selector-label {
  font-weight: 500;
  margin-right: 12px;
  color: var(--el-text-color-primary);
}

.fork-layout {
  display: grid;
  grid-template-columns: 1fr auto 1fr;
  gap: 20px;
  min-height: 500px;
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

.root-icon {
  font-size: 18px;
  color: var(--el-color-primary);
}

.root-label {
  font-size: 14px;
  font-weight: 600;
  color: var(--el-text-color-primary);
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
}

.tree-node-wrapper.is-drag-over {
  background: var(--el-color-primary-light-9);
  border: 2px dashed var(--el-color-primary);
}

.node-icon {
  font-size: 16px;
  color: var(--el-color-primary);
  flex-shrink: 0;
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
</style>
