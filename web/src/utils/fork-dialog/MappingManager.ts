/**
 * 映射关系管理器
 * 负责管理所有 Fork 映射关系，提供查询接口
 */

export interface ForkMapping {
  source: string      // 源函数组的 full_group_code: /user/app/package/group_code
  target: string      // 目标目录的 full_code_path: /user/app/package
  targetName?: string // 目标目录名称（显示用）
  sourceName?: string // 源函数组名称（显示用）
  colorIndex?: number // 颜色索引（标识同一次拖拽操作）
}

export interface MappingColor {
  bg: string
  border: string
  text: string
}

// 预定义的颜色方案
const COLOR_SCHEMES: MappingColor[] = [
  { bg: '#E3F2FD', border: '#2196F3', text: '#1976D2' },
  { bg: '#F3E5F5', border: '#9C27B0', text: '#7B1FA2' },
  { bg: '#E8F5E9', border: '#4CAF50', text: '#388E3C' },
  { bg: '#FFF3E0', border: '#FF9800', text: '#F57C00' },
  { bg: '#FCE4EC', border: '#E91E63', text: '#C2185B' },
  { bg: '#E0F2F1', border: '#009688', text: '#00796B' },
  { bg: '#F1F8E9', border: '#8BC34A', text: '#689F38' },
  { bg: '#E8EAF6', border: '#3F51B5', text: '#303F9F' },
]

export class MappingManager {
  private mappings: ForkMapping[] = []
  private nextColorIndex = 0
  
  // 索引结构，用于快速查询
  private byTarget = new Map<string, ForkMapping[]>()
  private bySource = new Map<string, ForkMapping[]>()
  private byColorIndex = new Map<number, ForkMapping[]>()
  
  // 源树引用，用于判断是否是函数组映射
  private sourceTree: any[] = []

  constructor(sourceTree: any[] = []) {
    this.sourceTree = sourceTree
  }

  /**
   * 设置源树（用于判断函数组映射）
   */
  setSourceTree(tree: any[]): void {
    this.sourceTree = tree
  }

  /**
   * 添加映射
   */
  addMapping(mapping: ForkMapping): void {
    // 检查是否已存在
    const exists = this.mappings.some(
      m => m.source === mapping.source && m.target === mapping.target
    )
    
    if (exists) {
      console.log('[MappingManager] 映射已存在，跳过:', mapping)
      return
    }

    // 如果没有颜色索引，分配一个
    if (mapping.colorIndex === undefined) {
      mapping.colorIndex = this.nextColorIndex++
    }

    console.log('[MappingManager] 添加映射:', {
      source: mapping.source,
      target: mapping.target,
      colorIndex: mapping.colorIndex,
      totalMappings: this.mappings.length + 1
    })

    this.mappings.push(mapping)
    this.updateIndexes(mapping)
  }

  /**
   * 批量添加映射
   */
  addMappings(mappings: ForkMapping[]): void {
    mappings.forEach(m => this.addMapping(m))
  }

  /**
   * 删除映射
   */
  removeMapping(source: string, target: string): void {
    const index = this.mappings.findIndex(
      m => m.source === source && m.target === target
    )
    
    if (index === -1) {
      console.log('[MappingManager] 映射不存在，无法删除:', { source, target })
      return
    }

    const mapping = this.mappings[index]
    console.log('[MappingManager] 删除映射:', {
      source: mapping.source,
      target: mapping.target,
      remainingMappings: this.mappings.length - 1
    })
    this.mappings.splice(index, 1)
    this.removeFromIndexes(mapping)
  }

  /**
   * 删除指定索引的映射
   */
  removeMappingByIndex(index: number): void {
    if (index < 0 || index >= this.mappings.length) {
      return
    }

    const mapping = this.mappings[index]
    this.mappings.splice(index, 1)
    this.removeFromIndexes(mapping)
  }

  /**
   * 清空所有映射
   */
  clear(): void {
    this.mappings = []
    this.byTarget.clear()
    this.bySource.clear()
    this.byColorIndex.clear()
  }

  /**
   * 获取所有映射
   */
  getAllMappings(): ForkMapping[] {
    return [...this.mappings]
  }

  /**
   * 根据 target 获取映射
   */
  getMappingsByTarget(target: string): ForkMapping[] {
    return this.byTarget.get(target) || []
  }

  /**
   * 根据 source 获取映射
   */
  getMappingsBySource(source: string): ForkMapping[] {
    return this.bySource.get(source) || []
  }

  /**
   * 根据颜色索引获取映射
   */
  getMappingsByColorIndex(colorIndex: number): ForkMapping[] {
    return this.byColorIndex.get(colorIndex) || []
  }

  /**
   * 获取节点相关的映射（包含精确匹配和前缀匹配）
   */
  getMappingsForNode(nodePath: string, isRootNode: boolean = false, rootPath?: string): ForkMapping[] {
    const result: ForkMapping[] = []
    
    // 精确匹配
    const exactMatches = this.getMappingsByTarget(nodePath)
    result.push(...exactMatches)

    // 前缀匹配：target 以 nodePath 开头
    this.byTarget.forEach((mappings, target) => {
      if (target.startsWith(nodePath + '/')) {
        result.push(...mappings)
      }
    })

    // 如果是根节点，也检查根路径格式
    if (isRootNode && rootPath) {
      const rootMatches = this.getMappingsByTarget(rootPath)
      rootMatches.forEach(m => {
        if (!result.some(r => r.source === m.source && r.target === m.target)) {
          result.push(m)
        }
      })
    }

    return result
  }

  /**
   * 判断 source 是否是函数组映射
   * 通过在源树中查找对应的节点来判断
   */
  isFunctionGroupMapping(source: string): boolean {
    // 方法1: 查找转换后的树（有 isGroup 标记）
    const findGroupNodeInTree = (nodes: any[], path: string): any | null => {
      for (const node of nodes) {
        // 检查是否是函数组节点且 full_group_code 匹配
        if (node.isGroup && node.full_group_code === path) {
          return node
        }
        // 递归查找子节点
        if (node.children && node.children.length > 0) {
          const found = findGroupNodeInTree(node.children, path)
          if (found) return found
        }
      }
      return null
    }

    // 方法2: 查找原始树中的函数节点（通过 full_group_code 匹配）
    const findFunctionNodeInTree = (nodes: any[], path: string): any | null => {
      for (const node of nodes) {
        // 如果是函数节点且 full_group_code 匹配
        if (node.type === 'function' && node.full_group_code === path) {
          return node
        }
        // 递归查找子节点
        if (node.children && node.children.length > 0) {
          const found = findFunctionNodeInTree(node.children, path)
          if (found) return found
        }
      }
      return null
    }

    // 先尝试在转换后的树中查找（如果有 isGroup 标记）
    let foundNode = findGroupNodeInTree(this.sourceTree, source)
    
    // 如果没找到，在原始树中查找函数节点
    if (!foundNode) {
      foundNode = findFunctionNodeInTree(this.sourceTree, source)
    }

    const isGroup = foundNode !== null
    console.log('[MappingManager] 判断函数组映射:', {
      source,
      foundNode: !!foundNode,
      isGroup,
      nodeType: foundNode?.type,
      nodeIsGroup: foundNode?.isGroup
    })
    return isGroup
  }

  /**
   * 区分函数组映射和目录映射
   */
  separateMappings(mappings: ForkMapping[]): {
    functionGroupMappings: ForkMapping[]
    directoryMappings: ForkMapping[]
  } {
    const functionGroupMappings: ForkMapping[] = []
    const directoryMappings: ForkMapping[] = []

    mappings.forEach(m => {
      if (this.isFunctionGroupMapping(m.source)) {
        functionGroupMappings.push(m)
      } else {
        directoryMappings.push(m)
      }
    })

    return { functionGroupMappings, directoryMappings }
  }

  /**
   * 获取下一个颜色索引
   */
  getNextColorIndex(): number {
    // 从所有映射中找出最大的 colorIndex
    const maxColorIndex = this.mappings.reduce((max, m) => {
      return Math.max(max, m.colorIndex ?? -1)
    }, -1)
    return maxColorIndex + 1
  }

  /**
   * 获取映射颜色
   */
  getMappingColor(mapping: ForkMapping | undefined | null): MappingColor {
    if (!mapping) {
      return COLOR_SCHEMES[0]
    }
    const colorIndex = mapping.colorIndex ?? 0
    return COLOR_SCHEMES[colorIndex % COLOR_SCHEMES.length]
  }

  /**
   * 更新索引
   */
  private updateIndexes(mapping: ForkMapping): void {
    // byTarget
    if (!this.byTarget.has(mapping.target)) {
      this.byTarget.set(mapping.target, [])
    }
    this.byTarget.get(mapping.target)!.push(mapping)

    // bySource
    if (!this.bySource.has(mapping.source)) {
      this.bySource.set(mapping.source, [])
    }
    this.bySource.get(mapping.source)!.push(mapping)

    // byColorIndex
    if (mapping.colorIndex !== undefined) {
      if (!this.byColorIndex.has(mapping.colorIndex)) {
        this.byColorIndex.set(mapping.colorIndex, [])
      }
      this.byColorIndex.get(mapping.colorIndex)!.push(mapping)
    }
  }

  /**
   * 从索引中移除
   */
  private removeFromIndexes(mapping: ForkMapping): void {
    // byTarget
    const targetMappings = this.byTarget.get(mapping.target)
    if (targetMappings) {
      const index = targetMappings.findIndex(
        m => m.source === mapping.source && m.target === mapping.target
      )
      if (index !== -1) {
        targetMappings.splice(index, 1)
        if (targetMappings.length === 0) {
          this.byTarget.delete(mapping.target)
        }
      }
    }

    // bySource
    const sourceMappings = this.bySource.get(mapping.source)
    if (sourceMappings) {
      const index = sourceMappings.findIndex(
        m => m.source === mapping.source && m.target === mapping.target
      )
      if (index !== -1) {
        sourceMappings.splice(index, 1)
        if (sourceMappings.length === 0) {
          this.bySource.delete(mapping.source)
        }
      }
    }

    // byColorIndex
    if (mapping.colorIndex !== undefined) {
      const colorMappings = this.byColorIndex.get(mapping.colorIndex)
      if (colorMappings) {
        const index = colorMappings.findIndex(
          m => m.source === mapping.source && m.target === mapping.target
        )
        if (index !== -1) {
          colorMappings.splice(index, 1)
          if (colorMappings.length === 0) {
            this.byColorIndex.delete(mapping.colorIndex)
          }
        }
      }
    }
  }
}

