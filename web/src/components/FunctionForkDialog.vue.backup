<script setup lang="ts">
import { ref, computed, watch, h, nextTick, onMounted, onUnmounted } from 'vue'
import { useRouter } from 'vue-router'
import { ElDialog, ElSelect, ElOption, ElButton, ElMessage, ElNotification, ElTag, ElEmpty, ElTree, ElForm, ElFormItem, ElDropdown, ElDropdownMenu, ElDropdownItem } from 'element-plus'
import { Delete, ArrowRight, Folder, FolderOpened, Plus, MoreFilled, Document } from '@element-plus/icons-vue'
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
  colorIndex?: number  // 颜色索引（用于标识同一拖拽操作）
}
const mappings = ref<ForkMapping[]>([])

// 颜色索引计数器（每次拖拽操作递增）
let nextColorIndex = 0

// 拖拽相关状态
const draggedNode = ref<ServiceTreeType | null>(null)
const dragOverNode = ref<ServiceTreeType | null>(null) // null 表示根目录
const isDragging = ref(false)

// 连接线相关状态
const connectionLines = ref<Array<{
  source: string
  target: string
  color: { bg: string; border: string; text: string }
  sourceRect?: DOMRect
  targetRect?: DOMRect
}>>([])
const sourceTreeRef = ref<HTMLElement | null>(null)
const targetTreeRef = ref<HTMLElement | null>(null)
const forkLayoutRef = ref<HTMLElement | null>(null)

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
  
  // 如果是目录，提前检查是否允许拖拽（防止拖拽到一半才发现不允许）
  if (isPackage) {
    // 检查：如果这个目录下的任何函数组或子目录已经被拖拽过，则不允许拖拽父目录
    const functionGroups = collectFunctionGroups(node)
    const alreadyMappedGroups = functionGroups.filter(group => {
      return mappings.value.some(m => m.source === group.full_group_code)
    })
    
    // 递归收集所有子目录（package类型）
    const collectSubPackages = (n: ServiceTreeType): ServiceTreeType[] => {
      const packages: ServiceTreeType[] = []
      if (n.type === 'package' && !(n as any).isGroup) {
        packages.push(n)
      }
      if (n.children && n.children.length > 0) {
        n.children.forEach(child => {
          packages.push(...collectSubPackages(child))
        })
      }
      return packages
    }
    
    const subPackages = collectSubPackages(node)
    // 检查子目录是否已经被拖拽过（通过检查子目录下的函数组是否在 mappings 中）
    const alreadyDraggedSubPackages = subPackages.filter(subPkg => {
      const subPkgGroups = collectFunctionGroups(subPkg)
      return subPkgGroups.some(group => {
        return mappings.value.some(m => m.source === group.full_group_code)
      })
    })
    
    if (alreadyMappedGroups.length > 0 || alreadyDraggedSubPackages.length > 0) {
      const conflictItems: string[] = []
      if (alreadyMappedGroups.length > 0) {
        conflictItems.push(`${alreadyMappedGroups.length}个函数组`)
      }
      if (alreadyDraggedSubPackages.length > 0) {
        conflictItems.push(`${alreadyDraggedSubPackages.length}个子目录`)
      }
      ElMessage.warning(`目录「${node.name}」下的${conflictItems.join('和')}已经被拖拽过，无法再次拖拽该目录。请拖拽其他目录或单独拖拽未拖拽的函数组。`)
      event.preventDefault()
      return false
    }
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
    // 为这次拖拽操作分配一个新的颜色索引
    const currentColorIndex = nextColorIndex++
    const newMapping: ForkMapping = {
      source: sourceNode.full_group_code,
      target: targetPath,
      targetName: targetName,
      sourceName: sourceNode.group_name || sourceNode.name || '',
      colorIndex: currentColorIndex
    }
    mappings.value.push(newMapping)
    
    ElMessage.success('已添加映射关系，点击"确认克隆"提交')
    
    // 更新连接线（延迟确保 DOM 更新）
    nextTick(() => {
      setTimeout(() => {
        updateConnectionLines()
      }, 150)
    })
  }
  // 情况2：拖拽服务目录
  else if (isSourcePackage) {
    // 检查：如果这个目录下的任何函数组或子目录已经被拖拽过，则不允许拖拽父目录
    // 1. 检查目录下的所有函数组
    const functionGroups = collectFunctionGroups(sourceNode)
    const alreadyMappedGroups = functionGroups.filter(group => {
      return mappings.value.some(m => m.source === group.full_group_code)
    })
    
    // 2. 检查子目录（package）是否已经被拖拽过
    // 递归收集所有子目录（package类型）
    const collectSubPackages = (node: ServiceTreeType): ServiceTreeType[] => {
      const packages: ServiceTreeType[] = []
      if (node.type === 'package' && !(node as any).isGroup) {
        packages.push(node)
      }
      if (node.children && node.children.length > 0) {
        node.children.forEach(child => {
          packages.push(...collectSubPackages(child))
        })
      }
      return packages
    }
    
    const subPackages = collectSubPackages(sourceNode)
    // 检查子目录是否已经被拖拽过（通过检查子目录下的函数组是否在 mappings 中）
    const alreadyDraggedSubPackages = subPackages.filter(subPkg => {
      const subPkgGroups = collectFunctionGroups(subPkg)
      return subPkgGroups.some(group => {
        return mappings.value.some(m => m.source === group.full_group_code)
      })
    })
    
    if (alreadyMappedGroups.length > 0 || alreadyDraggedSubPackages.length > 0) {
      const conflictItems: string[] = []
      if (alreadyMappedGroups.length > 0) {
        conflictItems.push(`${alreadyMappedGroups.length}个函数组`)
      }
      if (alreadyDraggedSubPackages.length > 0) {
        conflictItems.push(`${alreadyDraggedSubPackages.length}个子目录`)
      }
      ElMessage.warning(`目录「${sourceNode.name}」下的${conflictItems.join('和')}已经被拖拽过，无法再次拖拽该目录。请拖拽其他目录或单独拖拽未拖拽的函数组。`)
      dragOverNode.value = null
      draggedNode.value = null
      return
    }
    
    try {
      // 1. 在目标目录下递归创建目录结构（包括所有子目录）
      // 如果是根目录（node 为 null），parent_id 为 0；否则使用 node.id
      const parentId = node === null ? 0 : Number(node.id)
      const parentPath = node === null 
        ? `${selectedApp.value.user}/${selectedApp.value.code}`
        : node.full_code_path
      
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
      // 为这次拖拽操作分配一个新的颜色索引（目录下的所有函数组使用相同颜色）
      const currentColorIndex = nextColorIndex++
      const newMappings: typeof mappings.value = []
      
      // 创建一个辅助函数，根据源路径找到对应的目标路径（只到 package 目录）
      const findTargetPath = (sourceGroupCode: string): string => {
        // sourceGroupCode 格式: /user/app/path/to/package/group_code
        // 需要找到对应的目标路径（只到 package 目录，不包括 group_code）
        // 例如：源路径 /luobei/testfork666/tools/tools/tools_cashier
        //      目标路径应该是 /luobei/testfork777/tools/tools（去掉最后的 group_code）
        
        // 提取源路径中相对于源目录的部分
        const sourcePath = sourceNode.full_code_path
        // 标准化路径：确保都以 / 开头，且不以 / 结尾（除了根路径）
        const normalizedSourcePath = sourcePath.endsWith('/') && sourcePath !== '/' 
          ? sourcePath.slice(0, -1) 
          : sourcePath
        const normalizedSourceGroupCode = sourceGroupCode.startsWith('/') 
          ? sourceGroupCode 
          : '/' + sourceGroupCode
        
        // 检查 sourceGroupCode 是否以 sourcePath 开头（精确匹配或后面跟 /）
        if (normalizedSourceGroupCode === normalizedSourcePath || 
            normalizedSourceGroupCode.startsWith(normalizedSourcePath + '/')) {
          // 获取相对路径部分（包括 package 和 group_code）
          let relativePath = normalizedSourceGroupCode.substring(normalizedSourcePath.length)
          // 如果 relativePath 以 / 开头，去掉开头的 /
          if (relativePath.startsWith('/')) {
            relativePath = relativePath.substring(1)
          }
          
          // 去掉最后的 group_code 部分（最后一个路径段）
          // 例如：tools_cashier -> （空，因为函数组就在源目录下）
          const relativePathSegments = relativePath.split('/').filter(Boolean)
          if (relativePathSegments.length > 0) {
            // 去掉最后一个路径段（group_code），只保留 package 路径
            relativePathSegments.pop()
            const packageRelativePath = relativePathSegments.length > 0 
              ? '/' + relativePathSegments.join('/')
              : ''
            // 构建目标路径（只到 package 目录）
            // 注意：如果 packageRelativePath 为空，说明函数组就在源目录下，直接使用新创建的目录
            if (packageRelativePath) {
              return newDirectory.full_code_path + packageRelativePath
            } else {
              // 函数组就在源目录下，直接使用新创建的目录
              return newDirectory.full_code_path
            }
          }
          // 如果没有相对路径（函数组就在最外层目录下），直接使用新创建的目录
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
          console.log('[拖拽根目录] 函数组:', {
            sourceGroupCode: group.full_group_code,
            sourcePath: sourceNode.full_code_path,
            targetPath,
            newDirectoryPath: newDirectory.full_code_path
          })
          
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
              sourceName: group.group_name || group.name || '',
              colorIndex: currentColorIndex  // 同一目录下的所有函数组使用相同颜色
            })
          }
        }
      })
      
      console.log('[拖拽根目录] 新增映射数量:', newMappings.length, newMappings)
      
      if (newMappings.length > 0) {
        mappings.value.push(...newMappings)
        ElMessage.success(`已添加 ${newMappings.length} 个函数组的映射关系，点击"确认克隆"提交`)
        
        // 更新连接线（延迟确保 DOM 更新）
        nextTick(() => {
          setTimeout(() => {
            updateConnectionLines()
          }, 150)
        })
      } else {
        ElMessage.warning('所有函数组的映射关系已存在')
      }
      
      // 4. 刷新目标服务目录树，以便看到新创建的目录
      await loadTargetServiceTree(selectedApp.value)
      
      // 再次更新连接线（树更新后，延迟更长时间确保完全渲染）
      nextTick(() => {
        setTimeout(() => {
          updateConnectionLines()
        }, 300)
      })
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
              const url = `/workspace/${targetApp.user}/${targetApp.code}${forkedPaths ? `?forked=${encodeURIComponent(forkedPaths)}` : ''}`
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
  selectedApp.value = null
  sourceServiceTree.value = []
  targetServiceTree.value = []
  mappings.value = []
  nextColorIndex = 0  // 重置颜色索引计数器
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
  } else {
    connectionLines.value = []
  }
})

// 计算连接线路径（使用贝塞尔曲线）
const getConnectionPath = (sourceRect: DOMRect, targetRect: DOMRect): string => {
  // 确保坐标是有效数字
  const sourceX = (sourceRect.x || 0) + (sourceRect.width || 0)
  const sourceY = (sourceRect.y || 0) + ((sourceRect.height || 0) / 2)
  const targetX = targetRect.x || 0
  const targetY = (targetRect.y || 0) + ((targetRect.height || 0) / 2)
  
  // 检查是否为有效数字
  if (isNaN(sourceX) || isNaN(sourceY) || isNaN(targetX) || isNaN(targetY)) {
    console.warn('[连接线] 坐标包含 NaN:', { sourceRect, targetRect })
    return ''
  }
  
  // 计算控制点，让曲线更平滑
  const controlPoint1X = sourceX + (targetX - sourceX) * 0.5
  const controlPoint1Y = sourceY
  const controlPoint2X = sourceX + (targetX - sourceX) * 0.5
  const controlPoint2Y = targetY
  
  // 使用三次贝塞尔曲线
  return `M ${sourceX} ${sourceY} C ${controlPoint1X} ${controlPoint1Y}, ${controlPoint2X} ${controlPoint2Y}, ${targetX} ${targetY}`
}

// 转义 CSS 选择器中的特殊字符
const escapeSelector = (str: string): string => {
  return str.replace(/[!"#$%&'()*+,.\/:;<=>?@[\\\]^`{|}~]/g, '\\$&')
}

// 查找节点元素（支持多种查找方式）
const findNodeElement = (container: HTMLElement, identifier: string): HTMLElement | null => {
  // 方式1: 使用 data-node-id 属性（转义特殊字符）
  let element = container.querySelector(
    `[data-node-id="${escapeSelector(identifier)}"]`
  ) as HTMLElement

  if (element) return element

  // 方式2: 使用 data-node-id 属性（不转义，直接匹配）
  element = container.querySelector(
    `[data-node-id="${identifier}"]`
  ) as HTMLElement

  if (element) return element

  // 方式3: 查找所有带有 data-node-id 的元素，然后匹配
  const allNodes = container.querySelectorAll('[data-node-id]')
  for (const node of allNodes) {
    const nodeId = (node as HTMLElement).getAttribute('data-node-id')
    if (nodeId === identifier) {
      return node as HTMLElement
    }
  }

  return null
}

// 计算连接线
const updateConnectionLines = () => {
  if (!sourceTreeRef.value || !targetTreeRef.value || !forkLayoutRef.value || mappings.value.length === 0) {
    connectionLines.value = []
    return
  }

  const lines: typeof connectionLines.value = []
  const layoutRect = forkLayoutRef.value.getBoundingClientRect()

  // 按 colorIndex 分组映射，同一拖拽操作的映射使用相同的连接线起点（父目录）
  const mappingsByColorIndex = new Map<number, ForkMapping[]>()
  mappings.value.forEach(mapping => {
    if (mapping.colorIndex !== undefined) {
      if (!mappingsByColorIndex.has(mapping.colorIndex)) {
        mappingsByColorIndex.set(mapping.colorIndex, [])
      }
      mappingsByColorIndex.get(mapping.colorIndex)!.push(mapping)
    }
  })

  // 为每个 colorIndex 创建一条连接线（从父目录到目标目录）
  mappingsByColorIndex.forEach((colorMappings, colorIndex) => {
    if (colorMappings.length === 0) return

    const firstMapping = colorMappings[0]
    if (!firstMapping) return

    // 找到这些映射对应的源函数组路径
    const sourcePaths = colorMappings.map(m => m.source)
    
    // 找到共同的父目录路径（最长的公共前缀）
    const findCommonParentPath = (paths: string[]): string | null => {
      if (paths.length === 0) return null
      if (paths.length === 1) {
        // 如果是单个路径，提取父目录路径（去掉最后的 group_code）
        const parts = paths[0].split('/').filter(Boolean)
        if (parts.length > 3) {
          parts.pop() // 去掉 group_code
          return '/' + parts.join('/')
        }
        return null
      }
      
      // 找到所有路径的公共前缀
      const pathParts = paths.map(p => p.split('/').filter(Boolean))
      const minLength = Math.min(...pathParts.map(p => p.length))
      
      let commonParts: string[] = []
      for (let i = 0; i < minLength - 1; i++) { // -1 因为最后一个是 group_code
        const part = pathParts[0][i]
        if (pathParts.every(p => p[i] === part)) {
          commonParts.push(part)
        } else {
          break
        }
      }
      
      if (commonParts.length >= 2) { // 至少要有 user 和 app
        return '/' + commonParts.join('/')
      }
      return null
    }

    const commonParentPath = findCommonParentPath(sourcePaths)
    
    // 查找源节点（优先使用父目录，如果没有则使用第一个函数组）
    let sourceElement: HTMLElement | null = null
    if (commonParentPath) {
      // 尝试查找父目录节点
      sourceElement = findNodeElement(sourceTreeRef.value!, commonParentPath)
    }
    
    if (!sourceElement) {
      // 如果没有找到父目录，使用第一个函数组
      sourceElement = findNodeElement(sourceTreeRef.value!, firstMapping.source)
    }

    // 查找目标节点
    // 对于函数组映射，先尝试查找待克隆的函数组节点（使用 full_group_code）
    // 对于目录映射，直接查找目标目录
    let targetElement: HTMLElement | null = null
    
    // 判断是函数组映射还是目录映射（使用辅助函数）
    const isMappingFunctionGroup = isFunctionGroupMapping(firstMapping.source)
    
    if (isMappingFunctionGroup) {
      // 函数组映射：先查找待克隆的函数组节点
      console.log('[连接线] 查找函数组节点，source:', firstMapping.source)
      targetElement = findNodeElement(targetTreeRef.value!, firstMapping.source)
      console.log('[连接线] 函数组节点查找结果:', !!targetElement)
      
      // 如果没找到函数组节点，查找目标目录（函数组应该显示在目录下）
      if (!targetElement) {
        console.log('[连接线] 未找到函数组节点，查找目标目录:', firstMapping.target)
        targetElement = findNodeElement(targetTreeRef.value!, firstMapping.target)
        console.log('[连接线] 目标目录查找结果:', !!targetElement)
      }
    } else {
      // 目录映射：直接查找目标目录
      console.log('[连接线] 查找目录节点，target:', firstMapping.target)
      targetElement = findNodeElement(targetTreeRef.value!, firstMapping.target)
      console.log('[连接线] 目录节点查找结果:', !!targetElement)
    }

    // 如果还是没有找到，尝试查找父目录（向上查找）
    if (!targetElement) {
      let currentPath = firstMapping.target
      const pathParts = currentPath.split('/').filter(Boolean)
      console.log('[连接线] 未找到节点，尝试查找父目录，pathParts:', pathParts)
      
      // 从完整路径开始，逐步向上查找父目录
      while (pathParts.length > 2 && !targetElement) {
        // 移除最后一个路径段
        pathParts.pop()
        currentPath = '/' + pathParts.join('/')
        console.log('[连接线] 尝试查找父目录:', currentPath)
        targetElement = findNodeElement(targetTreeRef.value!, currentPath)
        if (targetElement) {
          console.log('[连接线] 找到父目录:', currentPath)
          break
        }
      }
      
      // 如果还是没找到，尝试查找根目录
      if (!targetElement && pathParts.length <= 2) {
        targetElement = findNodeElement(targetTreeRef.value!, 'root')
        console.log('[连接线] 根目录查找结果:', !!targetElement)
      }
    }

    if (sourceElement && targetElement) {
      const sourceRect = sourceElement.getBoundingClientRect()
      const targetRect = targetElement.getBoundingClientRect()

      // 检查元素是否可见（宽度和高度大于0）
      if (sourceRect.width > 0 && sourceRect.height > 0 && targetRect.width > 0 && targetRect.height > 0) {
        // 计算相对于 layout 的坐标
        const relativeSourceRect = {
          x: sourceRect.x - layoutRect.x,
          y: sourceRect.y - layoutRect.y,
          width: sourceRect.width,
          height: sourceRect.height,
          top: sourceRect.top - layoutRect.y,
          right: sourceRect.right - layoutRect.x,
          bottom: sourceRect.bottom - layoutRect.y,
          left: sourceRect.left - layoutRect.x
        } as DOMRect

        const relativeTargetRect = {
          x: targetRect.x - layoutRect.x,
          y: targetRect.y - layoutRect.y,
          width: targetRect.width,
          height: targetRect.height,
          top: targetRect.top - layoutRect.y,
          right: targetRect.right - layoutRect.x,
          bottom: targetRect.bottom - layoutRect.y,
          left: targetRect.left - layoutRect.x
        } as DOMRect

        // 验证坐标有效性
        if (!isNaN(relativeSourceRect.x) && !isNaN(relativeSourceRect.y) && 
            !isNaN(relativeTargetRect.x) && !isNaN(relativeTargetRect.y)) {
          // 使用第一个映射的颜色
          lines.push({
            source: commonParentPath || firstMapping.source,
            target: firstMapping.target,
            color: getMappingColor(firstMapping),
            sourceRect: relativeSourceRect,
            targetRect: relativeTargetRect
          })
        } else {
          console.warn('[连接线] 坐标无效:', {
            relativeSourceRect,
            relativeTargetRect,
            sourceRect,
            targetRect,
            layoutRect
          })
        }
      }
    }
  })

  connectionLines.value = lines
}



// 格式化显示路径（去掉用户名和应用名）
const formatPath = (path: string) => {
  const parts = path.split('/').filter(Boolean)
  if (parts.length >= 2) {
    return parts.slice(2).join('/')
  }
  return path
}

// 为映射生成颜色ID（基于source和target的hash）
const getMappingColorId = (source: string, target: string): number => {
  const combined = `${source}|${target}`
  let hash = 0
  for (let i = 0; i < combined.length; i++) {
    const char = combined.charCodeAt(i)
    hash = ((hash << 5) - hash) + char
    hash = hash & hash
  }
  return Math.abs(hash)
}

// 预定义的颜色方案（使用柔和的背景，深色的文字，增加到16种颜色）
const mappingColors = [
  { bg: '#fff0f6', border: '#ffadd2', text: '#722ed1' }, // 粉色背景，深紫色文字
  { bg: '#f6ffed', border: '#95de64', text: '#389e0d' }, // 绿色背景，深绿色文字
  { bg: '#e6f7ff', border: '#69c0ff', text: '#0050b3' }, // 蓝色背景，深蓝色文字
  { bg: '#fff7e6', border: '#ffc069', text: '#ad6800' }, // 橙色背景，深橙色文字
  { bg: '#f9f0ff', border: '#b37feb', text: '#531dab' }, // 紫色背景，深紫色文字
  { bg: '#fff1f0', border: '#ff7875', text: '#a8071a' }, // 红色背景，深红色文字
  { bg: '#e6fffb', border: '#5cdbd3', text: '#006d75' }, // 青色背景，深青色文字
  { bg: '#fffbe6', border: '#ffd666', text: '#ad6800' }, // 黄色背景，深橙色文字
  { bg: '#f0f5ff', border: '#adc6ff', text: '#2f54eb' }, // 浅蓝色背景，深蓝色文字
  { bg: '#f6ffed', border: '#b7eb8f', text: '#237804' }, // 浅绿色背景，深绿色文字
  { bg: '#fff2e8', border: '#ffbb96', text: '#d4380d' }, // 浅橙色背景，深橙色文字
  { bg: '#e6f7ff', border: '#91d5ff', text: '#0958d9' }, // 浅蓝色背景，深蓝色文字
  { bg: '#f0f5ff', border: '#d6e4ff', text: '#1d39c4' }, // 极浅蓝色背景，深蓝色文字
  { bg: '#fff1f0', border: '#ffccc7', text: '#cf1322' }, // 浅红色背景，深红色文字
  { bg: '#f6ffed', border: '#d9f7be', text: '#389e0d' }, // 极浅绿色背景，深绿色文字
  { bg: '#fff7e6', border: '#ffe7ba', text: '#d46b08' }, // 极浅橙色背景，深橙色文字
]

// 根据颜色索引获取颜色
const getColorByIndex = (colorIndex: number) => {
  return mappingColors[colorIndex % mappingColors.length]
}

// 获取映射对应的颜色（兼容旧代码）
const getMappingColor = (mapping: ForkMapping) => {
  if (mapping.colorIndex !== undefined) {
    return getColorByIndex(mapping.colorIndex)
  }
  // 如果没有 colorIndex，使用旧的 hash 方式（向后兼容）
  const colorId = getMappingColorId(mapping.source, mapping.target)
  return mappingColors[colorId % mappingColors.length]
}

// 查找节点对应的映射（用于源树）
const findSourceMapping = (node: ServiceTreeType): ForkMapping | null => {
  if ((node as any).isGroup && node.full_group_code) {
    return mappings.value.find(m => m.source === node.full_group_code) || null
  }
  // 如果是目录，检查目录下的所有函数组是否都在 mappings 中，并且它们有相同的 colorIndex
  if (node.type === 'package') {
    // 收集这个目录下的所有函数组（递归）
    const collectGroupsInNode = (n: ServiceTreeType): ServiceTreeType[] => {
      const groups: ServiceTreeType[] = []
      if ((n as any).isGroup && n.full_group_code) {
        groups.push(n)
      }
      if (n.children && n.children.length > 0) {
        n.children.forEach(child => {
          groups.push(...collectGroupsInNode(child))
        })
      }
      return groups
    }
    
    const groupsInNode = collectGroupsInNode(node)
    
    // 如果目录下有函数组，检查是否所有函数组都在 mappings 中，并且它们有相同的 colorIndex
    if (groupsInNode.length > 0) {
      // 找到所有函数组对应的映射
      const groupMappings = groupsInNode
        .map(group => {
          if (group.full_group_code) {
            return mappings.value.find(m => m.source === group.full_group_code)
          }
          return null
        })
        .filter((m): m is ForkMapping => m !== null)
      
      // 检查是否所有函数组都有映射
      if (groupMappings.length === groupsInNode.length && groupMappings.length > 0) {
        // 检查所有映射是否有相同的 colorIndex（说明是同一次拖拽操作）
        const firstMapping = groupMappings[0]
        if (firstMapping && firstMapping.colorIndex !== undefined) {
          const firstColorIndex = firstMapping.colorIndex
          const allSameColorIndex = groupMappings.every(m => m && m.colorIndex === firstColorIndex)
          
          // 只有当所有函数组都有映射且它们有相同的 colorIndex 时，才认为这个目录被拖拽了
          if (allSameColorIndex) {
            return firstMapping
          }
        }
      }
    }
  }
  return null
}

// 查找节点对应的映射（用于目标树）
const findTargetMapping = (node: ServiceTreeType): ForkMapping | null => {
  if ((node as any).isPending) {
    // 如果是待克隆的函数组，查找对应的映射
    if ((node as any).isGroup && node.full_group_code) {
      return mappings.value.find(m => m.source === node.full_group_code) || null
    }
    // 如果是待克隆的目录，查找以这个目录为target的映射
    const path = node.full_code_path
    return mappings.value.find(m => m.target === path || m.target.startsWith(path + '/')) || null
  }
  return null
}

// 将源服务目录树处理为带函数组的结构（参考 ServiceTreePanel 的逻辑）
const groupedSourceTree = computed(() => {
  // 先处理所有节点，构建完整的树结构
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
        
        // 检查这个函数组是否有对应的映射
        const mapping = mappings.value.find(m => m.source === groupCode)
        if (mapping) {
          (groupNode as any).mappingColor = getMappingColor(mapping)
        }
        
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
  
  // 先构建完整的树结构
  const processedTree = sourceServiceTree.value.map(node => processNode(node))
  
  // 然后为所有节点应用颜色
  // 先处理子节点，再处理父节点（这样父节点可以使用子节点的颜色信息）
  const applyColors = (node: ServiceTreeType): ServiceTreeType => {
    // 先递归处理子节点
    let nodeWithColor = {
      ...node,
      mappingColor: undefined
    } as ServiceTreeType & { mappingColor?: any }
    
    if (nodeWithColor.children && nodeWithColor.children.length > 0) {
      nodeWithColor.children = nodeWithColor.children.map(child => applyColors(child))
    }
    
    // 然后查找当前节点的映射
    const mapping = findSourceMapping(nodeWithColor)
    
    // 如果没有直接映射，检查是否有直接子节点被映射（用于父目录高亮）
    // 注意：只检查直接子节点中的函数组，不检查子目录，避免所有父目录都被高亮
    if (!mapping && nodeWithColor.children && nodeWithColor.children.length > 0) {
      // 只检查直接子节点中的函数组（不检查子目录）
      const directChildMappings = nodeWithColor.children
        .map(child => {
          // 只检查函数组，不检查目录
          if ((child as any).isGroup && child.full_group_code) {
            return mappings.value.find(m => m.source === child.full_group_code)
          }
          return null
        })
        .filter((m): m is ForkMapping => m !== null)
      
      // 如果有直接子节点中的函数组被映射，就高亮父目录（使用第一个映射的颜色）
      if (directChildMappings.length > 0) {
        const firstMapping = directChildMappings[0]
        if (firstMapping) {
          nodeWithColor.mappingColor = getMappingColor(firstMapping)
        }
      }
    } else if (mapping) {
      nodeWithColor.mappingColor = getMappingColor(mapping)
    }
    
    return nodeWithColor
  }
  
  return processedTree.map(node => applyColors(node))
})

// 检查目录是否被禁用（如果其子目录已经被拖拽过，则禁用）
const isDirectoryDisabled = (node: ServiceTreeType): boolean => {
  if (node.type !== 'package' || (node as any).isGroup) {
    return false
  }
  
  // 收集这个目录下的所有函数组
  const functionGroups = collectFunctionGroups(node)
  
  // 检查是否有任何函数组已经被拖拽过
  const hasMappedGroups = functionGroups.some(group => {
    return mappings.value.some(m => m.source === group.full_group_code)
  })
  
  return hasMappedGroups
}

// 检查 source 是否是函数组路径（在源树中查找对应的函数组节点）
const isFunctionGroupMapping = (source: string): boolean => {
  // 在源树中递归查找对应的节点
  const findNodeInSourceTree = (nodes: ServiceTreeType[], path: string): ServiceTreeType | null => {
    for (const node of nodes) {
      // 检查是否是函数组节点且 full_group_code 匹配
      if ((node as any).isGroup && node.full_group_code === path) {
        return node
      }
      // 递归查找子节点
      if (node.children && node.children.length > 0) {
        const found = findNodeInSourceTree(node.children, path)
        if (found) return found
      }
    }
    return null
  }
  
  const foundNode = findNodeInSourceTree(sourceServiceTree.value, source)
  const isGroup = foundNode !== null && (foundNode as any).isGroup === true
  console.log(`[isFunctionGroupMapping] source: ${source}, foundNode:`, !!foundNode, 'isGroup:', isGroup)
  return isGroup
}

// 将目标服务目录树处理为带函数组的结构（只显示 package 和函数组，不显示单个函数）
const groupedTargetTree = computed(() => {
  console.log('[groupedTargetTree] 开始处理，当前 mappings 数量:', mappings.value.length)
  console.log('[groupedTargetTree] mappings 详情:', mappings.value.map(m => ({ source: m.source, target: m.target })))
  
  // 创建一个映射，用于快速查找某个目录下已添加映射的函数组
  const mappingsByTarget = new Map<string, ForkMapping[]>()
  mappings.value.forEach(mapping => {
    if (!mappingsByTarget.has(mapping.target)) {
      mappingsByTarget.set(mapping.target, [])
    }
    mappingsByTarget.get(mapping.target)!.push(mapping)
  })
  
  console.log('[groupedTargetTree] mappingsByTarget keys:', Array.from(mappingsByTarget.keys()))
  
  
  const processNode = (node: ServiceTreeType): ServiceTreeType | null => {
    // 如果是函数节点，直接返回 null（不显示）
    if (node.type === 'function') {
      return null
    }
    
    // 检查这个目录是否有待克隆的映射（作为 target）
    // 注意：只有当目录本身被拖拽（即目录下的所有函数组都被拖拽）时，才标记目录为 isPending
    // 如果只是目录下的某些函数组被拖拽，目录不应该被标记为 isPending，只有那些函数组应该被标记
    const pendingMappingsForThisDir = mappingsByTarget.get(node.full_code_path) || []
    
    // 检查这些映射是否都是函数组映射（source 是 full_group_code，不是目录路径）
    // 如果都是函数组映射，说明目录只是作为目标，不应该标记为 isPending
    const allAreFunctionGroupMappings = pendingMappingsForThisDir.length > 0 && 
      pendingMappingsForThisDir.every(m => {
        // 使用辅助函数检查 source 是否是函数组路径
        return isFunctionGroupMapping(m.source)
      })
    
    // 只有当目录本身被拖拽（即不是所有映射都是函数组映射）时，才标记目录为 isPending
    const hasPendingMappings = pendingMappingsForThisDir.length > 0 && !allAreFunctionGroupMappings
    
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
      // 注意：需要查找所有以 node.full_code_path 为前缀的映射，因为函数组可能映射到子目录
      let pendingMappings = mappingsByTarget.get(node.full_code_path) || []
      console.log(`[groupedTargetTree] 目录 ${node.name} (${node.full_code_path}) 精确匹配的映射数量:`, pendingMappings.length)
      
      // 如果没找到精确匹配，也查找所有 target 以 node.full_code_path 开头的映射（说明映射到子目录）
      if (pendingMappings.length === 0) {
        mappingsByTarget.forEach((maps, target) => {
          if (target.startsWith(node.full_code_path + '/')) {
            console.log(`[groupedTargetTree] 找到前缀匹配的 target:`, target, 'node.full_code_path:', node.full_code_path)
            maps.forEach(m => {
              // 使用辅助函数检查是否是函数组映射
              const isGroup = isFunctionGroupMapping(m.source)
              console.log(`[groupedTargetTree] 前缀匹配检查: source=${m.source}, isFunctionGroup=${isGroup}`)
              if (isGroup) {
                console.log(`[groupedTargetTree] 添加函数组映射（前缀匹配）:`, m.source, '->', m.target)
                pendingMappings.push(m)
              }
            })
          }
        })
      }
      
      // 如果没找到精确匹配，查找所有 target 等于 node.full_code_path 的函数组映射
      // 注意：需要同时检查标准格式（/user/app/package）和根路径格式（user/app）
      if (pendingMappings.length === 0) {
        const allMappings: ForkMapping[] = []
        const isRootNode = node.parent_id === 0
        const rootPath = selectedApp.value ? `/${selectedApp.value.user}/${selectedApp.value.code}` : ''
        const rootPathAlt = selectedApp.value ? `${selectedApp.value.user}/${selectedApp.value.code}` : ''
        
        console.log(`[groupedTargetTree] 目录 ${node.name} 未找到精确匹配，开始查找。isRootNode:`, isRootNode, 'rootPath:', rootPath, 'rootPathAlt:', rootPathAlt, 'node.full_code_path:', node.full_code_path)
        
        mappingsByTarget.forEach((maps, target) => {
          // 精确匹配：target 等于 node.full_code_path
          // 或者，如果 node 是根目录，检查 target 是否匹配根路径格式
          // 或者，target 以 node.full_code_path 开头（说明映射到该目录或其子目录）
          const isExactMatch = target === node.full_code_path
          const isRootMatch = isRootNode && (target === rootPath || target === rootPathAlt)
          const isPrefixMatch = target.startsWith(node.full_code_path + '/') || node.full_code_path.startsWith(target + '/')
          
          if (isExactMatch || isRootMatch || isPrefixMatch) {
            console.log(`[groupedTargetTree] 找到匹配的 target:`, target, 'node.full_code_path:', node.full_code_path, 'isExactMatch:', isExactMatch, 'isRootMatch:', isRootMatch, 'isPrefixMatch:', isPrefixMatch, 'maps:', maps.length)
            maps.forEach(m => {
              // 使用辅助函数检查是否是函数组映射
              // 并且 target 应该等于 node.full_code_path（精确匹配）或者 target 以 node.full_code_path 开头
              const isGroup = isFunctionGroupMapping(m.source)
              console.log(`[groupedTargetTree] 匹配检查: source=${m.source}, target=${m.target}, isFunctionGroup=${isGroup}`)
              if (isGroup) {
                // 检查 target 是否匹配当前节点（精确匹配或前缀匹配）
                const targetMatches = m.target === node.full_code_path || m.target.startsWith(node.full_code_path + '/')
                console.log(`[groupedTargetTree] target匹配检查: targetMatches=${targetMatches}, node.full_code_path=${node.full_code_path}`)
                if (targetMatches) {
                  console.log(`[groupedTargetTree] 添加函数组映射:`, m.source, '->', m.target)
                  allMappings.push(m)
                }
              }
            })
          }
        })
        pendingMappings = allMappings
        console.log(`[groupedTargetTree] 目录 ${node.name} 最终找到的映射数量:`, pendingMappings.length)
      }
      
      // 区分函数组映射和目录映射
      const functionGroupMappings = pendingMappings.filter(m => {
        return isFunctionGroupMapping(m.source)
      })
      const directoryMappings = pendingMappings.filter(m => {
        return !isFunctionGroupMapping(m.source)
      })
      
      console.log(`[groupedTargetTree] 目录 ${node.name} 函数组映射数量:`, functionGroupMappings.length, '目录映射数量:', directoryMappings.length)
      
      functionGroupMappings.forEach(mapping => {
        // 检查是否已经存在对应的函数组节点（已 fork 的或已添加的）
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
          // 设置映射颜色
          pendingGroupNode.mappingColor = getMappingColor(mapping)
          
          newChildren.push(pendingGroupNode)
        }
      })
      
      // 只要有函数组映射或目录映射，就高亮目录
      const processedNode = {
        ...node,
        children: newChildren,
        // 只要有映射，就标记目录为 isPending
        isPending: pendingMappings.length > 0
      } as ServiceTreeType & { isPending?: boolean; mappingColor?: any }
      
      // 如果有待克隆映射，设置颜色（使用第一个映射的颜色）
      if (pendingMappings.length > 0) {
        const mapping = pendingMappings[0]
        if (mapping) {
          processedNode.mappingColor = getMappingColor(mapping)
        }
      }
      
      return processedNode
    }
    
    // 如果是 package 但没有子节点，也需要检查是否有待克隆的函数组
    if (node.type === 'package') {
      // 注意：需要查找所有以 node.full_code_path 为前缀的映射，因为函数组可能映射到子目录
      let pendingMappings = mappingsByTarget.get(node.full_code_path) || []
      
      // 如果没找到精确匹配，查找所有以该目录为前缀的映射（函数组可能映射到子目录）
      if (pendingMappings.length === 0) {
        const allMappings: ForkMapping[] = []
        mappingsByTarget.forEach((maps, target) => {
          if (target === node.full_code_path || target.startsWith(node.full_code_path + '/')) {
            // 检查这些映射是否真的是映射到这个目录或其子目录的函数组
            maps.forEach(m => {
              // 使用辅助函数检查是否是函数组映射，且 target 等于 node.full_code_path
              if (isFunctionGroupMapping(m.source) && m.target === node.full_code_path) {
                allMappings.push(m)
              }
            })
          }
        })
        pendingMappings = allMappings
      }
      
      if (pendingMappings.length > 0) {
        // 区分函数组映射和目录映射
        const functionGroupMappings = pendingMappings.filter(m => {
          return isFunctionGroupMapping(m.source)
        })
        
        const directoryMappings = pendingMappings.filter(m => {
          const sourceParts = m.source.split('/').filter(Boolean)
          const targetParts = m.target.split('/').filter(Boolean)
          return sourceParts.length <= targetParts.length
        })
        
        // 如果有函数组映射，需要创建 children 来显示它们
        if (functionGroupMappings.length > 0) {
          const newChildren: ServiceTreeType[] = []
          
          functionGroupMappings.forEach(mapping => {
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
              isPending: true,
              mappingColor: getMappingColor(mapping)
            } as ServiceTreeType & { isGroup?: boolean; isPending?: boolean; mappingColor?: any }
            
            newChildren.push(pendingGroupNode)
          })
          
          // 只要有函数组映射或目录映射，就高亮目录
          const processedNode = {
            ...node,
            children: newChildren,
            // 只要有映射，就标记目录为 isPending
            isPending: pendingMappings.length > 0,
            mappingColor: undefined
          } as ServiceTreeType & { isPending?: boolean; mappingColor?: any }
          
          // 如果有待克隆映射，设置颜色（使用第一个映射的颜色）
          if (pendingMappings.length > 0) {
            const mapping = pendingMappings[0]
            if (mapping) {
              processedNode.mappingColor = getMappingColor(mapping)
            }
          }
          
          return processedNode
        }
        
        // 如果只有目录映射（没有函数组映射），目录应该被标记为 isPending
        if (directoryMappings.length > 0) {
          const mapping = directoryMappings[0]
          const processedNode = {
            ...node,
            isPending: true
          } as ServiceTreeType & { isPending?: boolean; mappingColor?: any }
          
          if (mapping) {
            processedNode.mappingColor = getMappingColor(mapping)
          }
          
          return processedNode
        }
      }
    }
    
    // 如果是 package 但没有子节点且没有待克隆函数组，检查是否有待克隆映射
    if (node.type === 'package' && hasPendingMappings) {
      const mapping = mappingsByTarget.get(node.full_code_path)?.[0]
      const processedNode = {
        ...node,
        isPending: true
      } as ServiceTreeType & { isPending?: boolean; mappingColor?: any }
      
      if (mapping) {
        processedNode.mappingColor = getMappingColor(mapping)
      }
      
      return processedNode
    }
    
    // 如果是 package 但没有子节点且没有待克隆函数组，直接返回
    return node
  }
  
  const processedTree = targetServiceTree.value.map(node => processNode(node)).filter(node => node !== null) as ServiceTreeType[]
  
  // 4. 处理根目录下的待克隆函数组（如果目标路径是应用根路径）
  // 注意：需要同时检查标准格式（/user/app）和根路径格式（user/app）
  const rootPath1 = selectedApp.value ? `/${selectedApp.value.user}/${selectedApp.value.code}` : ''
  const rootPath2 = selectedApp.value ? `${selectedApp.value.user}/${selectedApp.value.code}` : ''
  const rootMappings = (rootPath1 ? mappingsByTarget.get(rootPath1) || [] : []).concat(
    rootPath2 ? mappingsByTarget.get(rootPath2) || [] : []
  )
  
  // 去重
  const uniqueRootMappings = rootMappings.filter((m, index, self) => 
    index === self.findIndex(n => n.source === m.source && n.target === m.target)
  )
  
  if (uniqueRootMappings.length > 0) {
    const rootPendingGroups: ServiceTreeType[] = []
    
    uniqueRootMappings.forEach(mapping => {
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
          full_code_path: `${rootPath1 || rootPath2}/__group__${groupCode}`,
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
    align-center
    @close="resetForm"
    class="fork-dialog"
  >
    <div class="fork-dialog-content">
      <!-- 顶部：选择目标应用 -->
      <div class="target-app-selector">
        <ElSelect
          v-model="selectedApp"
          placeholder="请选择目标应用"
          filterable
          :loading="loadingApps"
          style="width: 100%"
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
        
        <!-- 左侧：源应用的服务目录树（树形结构） -->
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
            >
              <template #default="{ node, data }">
                <div
                  :data-node-id="(data as any).isGroup && data.full_group_code ? data.full_group_code : (data.id || data.full_code_path)"
                  class="tree-node-wrapper"
                  :class="{
                    'is-draggable': ((data as any).isGroup && data.full_group_code && data.full_group_code.trim() !== '') || (data.type === 'package' && !(data as any).isGroup),
                    'is-dragging': draggedNode?.id === data.id,
                    'is-source': data.full_group_code === sourceFullGroupCode,
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
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup" class="node-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup" class="node-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <FolderOpened />
                  </el-icon>
                  <span v-else class="node-icon fx-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">fx</span>
                  <span class="node-label" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup && !(data as any).mappingColor" type="info" size="small" style="margin-left: 8px;">
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
                  <el-icon v-if="data.type === 'package' && !(data as any).isGroup" class="node-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <Folder />
                  </el-icon>
                  <el-icon v-else-if="(data as any).isGroup" class="node-icon" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">
                    <FolderOpened />
                  </el-icon>
                  <span class="node-label" :style="(data as any).mappingColor ? { color: (data as any).mappingColor.text } : {}">{{ node.label }}</span>
                  <ElTag v-if="(data as any).isGroup && !(data as any).isPending && !(data as any).mappingColor" type="info" size="small" style="margin-left: 8px;">
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
  z-index: 10;
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
