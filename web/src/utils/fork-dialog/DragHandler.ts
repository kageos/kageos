/**
 * 拖拽处理器
 * 负责处理拖拽操作，建立映射关系
 */

import type { ServiceTree as ServiceTreeType, App } from '@/types'
import { MappingManager, type ForkMapping } from './MappingManager'
import { createServiceTree } from '@/api/service-tree'

export class DragHandler {
  constructor(
    private mappingManager: MappingManager,
    private targetApp: App | null
  ) {}

  /**
   * 设置目标应用
   */
  setTargetApp(app: App | null): void {
    this.targetApp = app
  }

  /**
   * 验证是否可以拖拽节点
   */
  canDrag(node: ServiceTreeType): { allowed: boolean; reason?: string } {
    const isGroup = (node as any).isGroup && node.full_group_code
    const isPackage = node.type === 'package' && !(node as any).isGroup

    if (!isGroup && !isPackage) {
      return { allowed: false, reason: '只能拖拽函数组或服务目录' }
    }

    // 如果是目录，检查是否允许拖拽
    if (isPackage) {
      return this.canDragDirectory(node)
    }

    return { allowed: true }
  }

  /**
   * 检查是否可以拖拽目录
   */
  private canDragDirectory(node: ServiceTreeType): { allowed: boolean; reason?: string } {
    // 检查目录下的函数组是否已经被映射
    const functionGroups = this.collectFunctionGroups(node)
    const alreadyMappedGroups = functionGroups.filter(group => {
      return this.mappingManager.getMappingsBySource(group.full_group_code || '').length > 0
    })

    if (alreadyMappedGroups.length > 0) {
      return {
        allowed: false,
        reason: `目录「${node.name}」下的 ${alreadyMappedGroups.length} 个函数组已经被拖拽过，无法再次拖拽该目录`
      }
    }

    // 检查子目录是否已经被拖拽过
    const subPackages = this.collectSubPackages(node)
    const alreadyDraggedSubPackages = subPackages.filter(subPkg => {
      const subPkgGroups = this.collectFunctionGroups(subPkg)
      return subPkgGroups.some(group => {
        return this.mappingManager.getMappingsBySource(group.full_group_code || '').length > 0
      })
    })

    if (alreadyDraggedSubPackages.length > 0) {
      return {
        allowed: false,
        reason: `目录「${node.name}」下的 ${alreadyDraggedSubPackages.length} 个子目录已经被拖拽过，无法再次拖拽该目录`
      }
    }

    return { allowed: true }
  }

  /**
   * 处理函数组拖拽
   */
  handleGroupDrag(
    sourceNode: ServiceTreeType,
    targetNode: ServiceTreeType | null
  ): { success: boolean; message?: string } {
    console.log('[DragHandler] 处理函数组拖拽:', {
      sourceGroupCode: sourceNode.full_group_code,
      targetNode: targetNode ? targetNode.full_code_path : '根目录',
      targetApp: this.targetApp ? `${this.targetApp.user}/${this.targetApp.code}` : null
    })

    if (!this.targetApp) {
      return { success: false, message: '请先选择目标应用' }
    }

    if (!sourceNode.full_group_code) {
      return { success: false, message: '源函数组没有 full_group_code，无法克隆' }
    }

    // 计算目标路径
    const targetPath = targetNode === null
      ? `/${this.targetApp.user}/${this.targetApp.code}`
      : targetNode.full_code_path
    const targetName = targetNode === null ? '根目录' : targetNode.name

    // 检查是否已经存在相同的映射
    const existingMapping = this.mappingManager.getAllMappings().find(
      m => m.source === sourceNode.full_group_code && m.target === targetPath
    )

    if (existingMapping) {
      console.log('[DragHandler] 映射已存在:', existingMapping)
      return { success: false, message: '该映射关系已存在' }
    }

    // 创建映射（使用新的颜色索引）
    const mapping: ForkMapping = {
      source: sourceNode.full_group_code,
      target: targetPath,
      targetName,
      sourceName: (sourceNode as any).group_name || sourceNode.name || '',
      colorIndex: this.mappingManager.getNextColorIndex() // 使用新的颜色索引
    }

    console.log('[DragHandler] 创建函数组映射:', mapping)
    this.mappingManager.addMapping(mapping)

    return { success: true, message: '已添加映射关系，点击"确认克隆"提交' }
  }

  /**
   * 处理目录拖拽
   */
  async handleDirectoryDrag(
    sourceNode: ServiceTreeType,
    targetNode: ServiceTreeType | null
  ): Promise<{ success: boolean; message?: string; error?: any }> {
    console.log('[DragHandler] 处理目录拖拽:', {
      sourcePath: sourceNode.full_code_path,
      targetNode: targetNode ? targetNode.full_code_path : '根目录',
      targetApp: this.targetApp ? `${this.targetApp.user}/${this.targetApp.code}` : null
    })

    if (!this.targetApp) {
      return { success: false, message: '请先选择目标应用' }
    }

    try {
      // 1. 创建目录结构
      const parentId = targetNode === null ? 0 : Number(targetNode.id)
      const parentPath = targetNode === null
        ? `${this.targetApp.user}/${this.targetApp.code}`
        : targetNode.full_code_path

      console.log('[DragHandler] 开始创建目录结构，parentPath:', parentPath)
      const newDirectory = await this.createDirectoryRecursively(
        sourceNode,
        parentId,
        parentPath
      )
      console.log('[DragHandler] 目录创建完成:', newDirectory.full_code_path)

      // 2. 为新创建的目录创建目录映射（用于高亮显示）
      // 计算新目录的完整路径
      const newDirectoryPath = newDirectory.full_code_path
      const directoryMapping: ForkMapping = {
        source: sourceNode.full_code_path, // 源目录路径
        target: newDirectoryPath, // 新创建的目录路径
        targetName: newDirectory.name,
        sourceName: sourceNode.name || '',
        colorIndex: this.mappingManager.getNextColorIndex()
      }
      console.log('[DragHandler] 创建目录映射用于高亮:', directoryMapping)
      this.mappingManager.addMapping(directoryMapping)

      // 3. 收集所有函数组
      const functionGroups = this.collectFunctionGroups(sourceNode)
      console.log('[DragHandler] 收集到函数组数量:', functionGroups.length)

      if (functionGroups.length === 0) {
        return { success: false, message: '该目录下没有函数组' }
      }

      // 4. 为每个函数组创建映射（使用与目录映射相同的颜色索引）
      const newMappings: ForkMapping[] = []
      // 使用目录映射的颜色索引，这样目录和函数组使用相同的颜色
      const currentColorIndex = directoryMapping.colorIndex!

      functionGroups.forEach(group => {
        if (group.full_group_code) {
          // 计算目标路径
          const targetPath = this.calculateTargetPath(
            group.full_group_code,
            sourceNode.full_code_path,
            newDirectory.full_code_path
          )

          // 检查是否已存在
          const exists = this.mappingManager.getAllMappings().some(
            m => m.source === group.full_group_code && m.target === targetPath
          )

          if (!exists) {
            const targetPathSegments = targetPath.split('/').filter(Boolean)
            const targetDirName = targetPathSegments.length > 2
              ? targetPathSegments[targetPathSegments.length - 2]
              : newDirectory.name

            newMappings.push({
              source: group.full_group_code,
              target: targetPath,
              targetName: targetDirName,
              sourceName: (group as any).group_name || group.name || '',
              colorIndex: currentColorIndex
            })
          }
        }
      })

      console.log('[DragHandler] 准备添加映射，数量:', newMappings.length, newMappings)
      if (newMappings.length > 0) {
        this.mappingManager.addMappings(newMappings)
        return {
          success: true,
          message: `已添加 ${newMappings.length} 个函数组的映射关系，点击"确认克隆"提交`
        }
      }

      return { success: false, message: '没有可添加的映射关系' }
    } catch (error: any) {
      console.error('[DragHandler] 目录拖拽失败:', error)
      return {
        success: false,
        message: error?.message || '创建目录失败',
        error
      }
    }
  }

  /**
   * 递归创建目录结构
   */
  private async createDirectoryRecursively(
    sourceNode: ServiceTreeType,
    targetParentId: number,
    targetParentPath: string
  ): Promise<ServiceTreeType> {
    // 创建当前目录
    const newDirectory = await createServiceTree({
      user: this.targetApp!.user,
      app: this.targetApp!.code,
      name: sourceNode.name,
      code: sourceNode.code,
      parent_id: targetParentId,
      description: sourceNode.description || '',
      tags: sourceNode.tags || ''
    })

    // 递归创建子目录
    if (sourceNode.children && sourceNode.children.length > 0) {
      const packageChildren = sourceNode.children.filter(
        child => child.type === 'package' && !(child as any).isGroup
      )

      for (const childPackage of packageChildren) {
        await this.createDirectoryRecursively(
          childPackage,
          newDirectory.id,
          newDirectory.full_code_path
        )
      }
    }

    return newDirectory
  }

  /**
   * 计算函数组的目标路径
   */
  private calculateTargetPath(
    groupCode: string,
    sourcePath: string,
    newDirectoryPath: string
  ): string {
    // 如果 groupCode 以 sourcePath 开头，提取相对路径
    const normalizedGroupCode = groupCode.replace(/^\/+/, '')
    const normalizedSourcePath = sourcePath.replace(/^\/+/, '')

    if (normalizedGroupCode.startsWith(normalizedSourcePath + '/')) {
      // 获取相对路径部分
      let relativePath = normalizedGroupCode.substring(normalizedSourcePath.length)
      if (relativePath.startsWith('/')) {
        relativePath = relativePath.substring(1)
      }

      // 去掉最后的 group_code 部分
      const relativePathSegments = relativePath.split('/').filter(Boolean)
      if (relativePathSegments.length > 0) {
        relativePathSegments.pop() // 去掉 group_code
        const packageRelativePath = relativePathSegments.length > 0
          ? '/' + relativePathSegments.join('/')
          : ''
        
        if (packageRelativePath) {
          return newDirectoryPath + packageRelativePath
        } else {
          return newDirectoryPath
        }
      }
    }

    // 如果无法匹配，从 groupCode 中提取路径
    const segments = groupCode.split('/').filter(Boolean)
    if (segments.length > 0) {
      segments.pop() // 去掉最后的 group_code
      const packageSegments = segments.slice(2) // 去掉 user 和 app
      return `/${this.targetApp!.user}/${this.targetApp!.code}${packageSegments.length > 0 ? '/' + packageSegments.join('/') : ''}`
    }

    return newDirectoryPath
  }

  /**
   * 收集目录下的所有函数组
   */
  private collectFunctionGroups(node: ServiceTreeType): ServiceTreeType[] {
    const groups: ServiceTreeType[] = []

    if ((node as any).isGroup && node.full_group_code) {
      groups.push(node)
    }

    if (node.children && node.children.length > 0) {
      node.children.forEach(child => {
        groups.push(...this.collectFunctionGroups(child))
      })
    }

    return groups
  }

  /**
   * 收集所有子目录
   */
  private collectSubPackages(node: ServiceTreeType): ServiceTreeType[] {
    const packages: ServiceTreeType[] = []

    if (node.type === 'package' && !(node as any).isGroup) {
      packages.push(node)
    }

    if (node.children && node.children.length > 0) {
      node.children.forEach(child => {
        packages.push(...this.collectSubPackages(child))
      })
    }

    return packages
  }

  /**
   * 获取下一个颜色索引
   */
  private getNextColorIndex(): number {
    // 从所有映射中找出最大的 colorIndex
    const allMappings = this.mappingManager.getAllMappings()
    const maxColorIndex = allMappings.reduce((max, m) => {
      return Math.max(max, m.colorIndex ?? -1)
    }, -1)
    
    return maxColorIndex + 1
  }
}

