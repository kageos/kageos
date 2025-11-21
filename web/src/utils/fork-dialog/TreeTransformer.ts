/**
 * 树结构转换器
 * 负责将原始树结构转换为显示用的树结构
 */

import type { ServiceTree as ServiceTreeType } from '@/types'
import { createGroupNode, groupFunctionsByCode, getGroupName } from '@/utils/tree-utils'
import { MappingManager, type ForkMapping } from './MappingManager'

export interface NodeMetadata {
  hasMappings: boolean
  functionGroupMappings: ForkMapping[]
  directoryMappings: ForkMapping[]
  isPending: boolean
  mappingColor?: { bg: string; border: string; text: string }
}

export class TreeTransformer {
  constructor(
    private mappingManager: MappingManager
  ) {}

  /**
   * 转换源树（显示函数组）
   * 源树需要显示所有函数组，并标记已映射的函数组
   */
  transformSourceTree(tree: ServiceTreeType[]): ServiceTreeType[] {
    console.log('[TreeTransformer] 开始转换源树，节点数:', tree.length)
    const result = tree.map(node => this.transformSourceNode(node))
    console.log('[TreeTransformer] 源树转换完成')
    return result
  }

  /**
   * 转换源节点
   */
  private transformSourceNode(node: ServiceTreeType): ServiceTreeType {
    // 如果是函数节点，直接返回
    if (node.type === 'function') {
      return node
    }

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
        newChildren.push(this.transformSourceNode(pkg))
      })

      // 2. 添加分组后的函数（创建虚拟函数组节点）
      groupedFunctions.forEach((funcs, groupCode) => {
        const groupName = getGroupName(funcs, groupCode)
        const groupNode = createGroupNode(groupCode, groupName, node, true)
        // 函数组下包含函数节点
        groupNode.children = funcs.map(func => this.transformSourceNode(func))

        // 检查这个函数组是否有对应的映射
        const mappings = this.mappingManager.getMappingsBySource(groupCode)
        if (mappings.length > 0 && mappings[0]) {
          const mapping = mappings[0]
          try {
            console.log('[TreeTransformer] 函数组有映射，设置颜色:', groupCode, mapping.source, mapping.target, mapping.colorIndex)
          } catch (e) {
            // 忽略日志错误
          }
          (groupNode as any).mappingColor = this.mappingManager.getMappingColor(mapping)
        }

        newChildren.push(groupNode)
      })

      // 3. 添加未分组的函数
      ungroupedFunctions.forEach(func => {
        newChildren.push(this.transformSourceNode(func))
      })

      // 检查父目录是否有子节点被映射（用于高亮父目录）
      const directChildMappings = newChildren
        .map(child => {
          if ((child as any).isGroup && child.full_group_code) {
            return this.mappingManager.getMappingsBySource(child.full_group_code)[0]
          }
          return null
        })
        .filter((m): m is ForkMapping => m !== null)

      const processedNode: ServiceTreeType & { mappingColor?: any } = {
        ...node,
        children: newChildren
      }

      // 如果有直接子节点中的函数组被映射，就高亮父目录
      if (directChildMappings.length > 0 && directChildMappings[0]) {
        processedNode.mappingColor = this.mappingManager.getMappingColor(directChildMappings[0])
      }

      return processedNode
    }

    // 如果是函数或没有子节点，直接返回
    return node
  }

  /**
   * 转换目标树（显示目录和待 Fork 的函数组）
   * 目标树需要：
   * 1. 只显示 package 和函数组，不显示单个函数
   * 2. 显示待 Fork 的函数组（虚拟节点）
   * 3. 标记有映射的目录
   */
  transformTargetTree(
    tree: ServiceTreeType[],
    rootPath: string
  ): ServiceTreeType[] {
    console.log('[TreeTransformer] 开始转换目标树，节点数:', tree.length, 'rootPath:', rootPath)
    const result = tree.map(node => this.transformTargetNode(node, rootPath)).filter(node => node !== null) as ServiceTreeType[]
    console.log('[TreeTransformer] 目标树转换完成，结果节点数:', result.length)
    return result
  }

  /**
   * 转换目标节点
   */
  private transformTargetNode(
    node: ServiceTreeType,
    rootPath: string
  ): ServiceTreeType | null {
    // 如果是函数节点，不显示
    if (node.type === 'function') {
      return null
    }

    // 如果是 package 节点，需要处理
    if (node.type === 'package') {
      return this.processPackageNode(node, rootPath)
    }

    return node
  }

  /**
   * 处理 package 节点
   */
  private processPackageNode(
    node: ServiceTreeType,
    rootPath: string
  ): ServiceTreeType {
    const isRootNode = node.parent_id === 0
    const pendingMappings = this.mappingManager.getMappingsForNode(
      node.full_code_path,
      isRootNode,
      rootPath
    )

    try {
      console.log('[TreeTransformer] 处理 package 节点:', node.name, node.full_code_path, isRootNode, pendingMappings.length)
    } catch (e) {
      // 忽略日志错误
    }

    // 分离函数组映射和目录映射
    const { functionGroupMappings, directoryMappings } =
      this.mappingManager.separateMappings(pendingMappings)

    try {
      console.log('[TreeTransformer] 映射分离结果:', functionGroupMappings.length, directoryMappings.length)
    } catch (e) {
      // 忽略日志错误
    }

    // 构建新的 children 数组
    const newChildren: ServiceTreeType[] = []

    // 1. 处理子 package 节点
    if (node.children) {
      const packageChildren = node.children.filter(
        child => child.type === 'package' && !(child as any).isGroup
      )

      packageChildren.forEach(pkg => {
        const processed = this.transformTargetNode(pkg, rootPath)
        if (processed) {
          newChildren.push(processed)
        }
      })

      // 2. 处理函数，按 full_group_code 分组
      const functions = node.children.filter(child => child.type === 'function')
      const { grouped: groupedFunctions } = groupFunctionsByCode(functions)

      groupedFunctions.forEach((funcs, groupCode) => {
        const groupName = getGroupName(funcs, groupCode)
        const groupNode = createGroupNode(groupCode, groupName, node, false)
        newChildren.push(groupNode)
      })
    }

    // 3. 添加待 Fork 的函数组（虚拟节点）
    // 注意：函数组节点应该显示在目标目录下（mapping.target）
    // 例如：如果 target 是 /user/app/a/b/c，那么函数组应该显示在 /user/app/a/b/c 下
    functionGroupMappings.forEach(mapping => {
      // 只有当当前节点就是目标目录本身时，才创建函数组节点
      const isExactMatch = mapping.target === node.full_code_path
      
      if (!isExactMatch) {
        // 当前节点不是目标目录，跳过
        // 但是，如果目标目录是新创建的（还没有在树中），我们需要在目标目录的父目录下创建虚拟节点
        // 检查当前节点是否是目标目录的父目录
        const targetPathParts = mapping.target.split('/').filter(Boolean)
        const targetParentPath = '/' + targetPathParts.slice(0, -1).join('/')
        const isTargetParent = targetParentPath === node.full_code_path
        
        if (!isTargetParent) {
          return
        }
        
        // 当前节点是目标目录的父目录，但目标目录可能还没有在树中
        // 检查目标目录是否已经在子节点中
        const targetDirName = targetPathParts[targetPathParts.length - 1]
        const targetDirExists = newChildren.some(
          child => child.type === 'package' && !(child as any).isGroup && child.code === targetDirName
        )
        
        if (targetDirExists) {
          // 目标目录已存在，函数组应该显示在目标目录下，而不是父目录下
          return
        }
        
        // 目标目录不存在，说明是新创建的目录，但还没有在树中
        // 在这种情况下，我们在父目录下创建一个虚拟的函数组节点
        // 但这不是理想的做法，更好的做法是等待树刷新后再显示
        // 暂时跳过，等待树刷新后会自动显示
        return
      }

      // 检查是否已经存在对应的函数组节点
      const existingGroup = newChildren.find(
        child => (child as any).isGroup && child.full_group_code === mapping.source
      )

      // 如果不存在，创建虚拟函数组节点
      if (!existingGroup) {
        const groupCode = mapping.source
        const groupName = mapping.sourceName || getGroupName([], groupCode)

        try {
          console.log('[TreeTransformer] 创建待 Fork 函数组节点:', {
            groupCode,
            groupName,
            parentNode: node.name,
            parentPath: node.full_code_path,
            target: mapping.target,
            isExactMatch
          })
        } catch (e) {
          // 忽略日志错误
        }

        const pendingGroupNode = createGroupNode(groupCode, groupName, node, false)
        pendingGroupNode.isPending = true
        pendingGroupNode.mappingColor = this.mappingManager.getMappingColor(mapping)

        newChildren.push(pendingGroupNode)
      } else {
        try {
          console.log('[TreeTransformer] 函数组节点已存在，跳过:', mapping.source)
        } catch (e) {
          // 忽略日志错误
        }
      }
    })

    // 4. 构建处理后的节点
    // 注意：只有精确匹配的目录映射才应该高亮目录本身
    // 前缀匹配的映射只用于在子目录中显示待 Fork 的函数组
    const exactDirectoryMappings = directoryMappings.filter(m => m.target === node.full_code_path)
    
    const processedNode: ServiceTreeType & {
      isPending?: boolean
      mappingColor?: { bg: string; border: string; text: string }
    } = {
      ...node,
      children: newChildren,
      // 只有精确匹配的目录映射才标记为 isPending
      isPending: exactDirectoryMappings.length > 0
    }

    // 只有精确匹配的目录映射才设置颜色
    if (exactDirectoryMappings.length > 0 && exactDirectoryMappings[0]) {
      processedNode.mappingColor = this.mappingManager.getMappingColor(exactDirectoryMappings[0])
    }

    return processedNode
  }

  /**
   * 获取节点的元数据
   */
  getNodeMetadata(
    node: ServiceTreeType,
    rootPath: string
  ): NodeMetadata {
    const isRootNode = node.parent_id === 0
    const pendingMappings = this.mappingManager.getMappingsForNode(
      node.full_code_path,
      isRootNode,
      rootPath
    )

    const { functionGroupMappings, directoryMappings } =
      this.mappingManager.separateMappings(pendingMappings)

    return {
      hasMappings: pendingMappings.length > 0,
      functionGroupMappings,
      directoryMappings,
      isPending: pendingMappings.length > 0,
      mappingColor: pendingMappings.length > 0 && pendingMappings[0]
        ? this.mappingManager.getMappingColor(pendingMappings[0])
        : undefined
    }
  }
}

